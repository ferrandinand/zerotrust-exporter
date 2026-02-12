package devices

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/vinistoisr/zerotrust-exporter/internal/appmetrics"
	"github.com/vinistoisr/zerotrust-exporter/internal/config"
)

// PhysicalDevice represents a physical device from the Cloudflare API
type PhysicalDevice struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	DeviceType          string    `json:"device_type"`
	Model               string    `json:"model"`
	OSVersion           string    `json:"os_version"`
	SerialNumber        string    `json:"serial_number"`
	MacAddress          string    `json:"mac_address"`
	ClientVersion       string    `json:"client_version"`
	PublicIP            string    `json:"public_ip"`
	ActiveRegistrations int       `json:"active_registrations"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	LastSeenAt          time.Time `json:"last_seen_at"`
	LastSeenUser        struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"last_seen_user"`
}

// DeviceStatus is a simplified struct for internal use
type DeviceStatus struct {
	DeviceID    string
	DeviceName  string
	DeviceType  string
	OSVersion   string
	Version     string
	PersonEmail string
	PublicIP    string
}

func fetchDeviceStatus(ctx context.Context, accountID string) (map[string]DeviceStatus, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/devices/physical-devices", accountID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		appmetrics.IncApiErrorsCounter()
		return nil, err
	}
	// add authorization headers
	req.Header.Set("Authorization", "Bearer "+config.ApiKey)
	req.Header.Set("Content-Type", "application/json")
	// send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	// defer closing the response body
	defer resp.Body.Close()
	// increment the api call counter
	appmetrics.IncApiCallCounter()
	// parse the status code if not ok
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		return nil, fmt.Errorf("failed to fetch device status: %s, response body: %s", resp.Status, bodyString)
	}
	// parse the response body into a struct
	var response struct {
		Result []PhysicalDevice `json:"result"`
	}
	// decode the response body
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	deviceStatuses := make(map[string]DeviceStatus)
	for _, device := range response.Result {
		deviceStatuses[device.ID] = DeviceStatus{
			DeviceID:    device.ID,
			DeviceName:  device.Name,
			DeviceType:  device.DeviceType,
			OSVersion:   device.OSVersion,
			Version:     device.ClientVersion,
			PersonEmail: device.LastSeenUser.Email,
			PublicIP:    device.PublicIP,
		}
	}

	return deviceStatuses, nil
}

func CollectDeviceMetrics() map[string]DeviceStatus {
	appmetrics.IncApiCallCounter()
	ctx := context.Background()
	startTime := time.Now()

	deviceStatuses, err := fetchDeviceStatus(ctx, config.AccountID)
	if err != nil {
		log.Printf("Error fetching device status: %v", err)
		appmetrics.IncApiErrorsCounter()
		return nil
	}

	if config.Debug {
		log.Printf("Fetched %d devices in %v", len(deviceStatuses), time.Since(startTime))
	}

	// All devices from the physical-devices endpoint are considered active
	for deviceID, status := range deviceStatuses {
		metricName := fmt.Sprintf(`zerotrust_devices_up{device_id="%s", device_name="%s", user_email="%s", device_type="%s", os_version="%s", version="%s", public_ip="%s"}`,
			deviceID, status.DeviceName, status.PersonEmail, status.DeviceType, status.OSVersion, status.Version, status.PublicIP)
		gauge := metrics.GetOrCreateGauge(metricName, nil)
		gauge.Set(1)
	}

	log.Println("Device metrics collection completed.")
	return deviceStatuses
}
