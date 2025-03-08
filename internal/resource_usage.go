package internal

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

// ProcessResourceUsage represents a single process entry returned by the Agent API endpoint.
type ProcessResourceUsage struct {
	PID           int     `json:"pid"`
	MemoryPercent float64 `json:"memory_percent"`
	CPUPercent    float64 `json:"cpu_percent"`
	Status        string  `json:"status"`
	Name          string  `json:"name"`
	CreateTime    float64 `json:"create_time"`
	Username      string  `json:"username"`
}

func ServerResourceDetails(baseURL string, limit int) ([]ProcessResourceUsage, error) {
	// Construct URL with query parameter.
	resp, err := http.Get(baseURL + "/api/v1/agent/resource-usage?limit=" + strconv.Itoa(limit))

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse JSON response.
	var processes []ProcessResourceUsage
	if err := json.Unmarshal(body, &processes); err != nil {
		return nil, err
	}

	return processes, nil
}
