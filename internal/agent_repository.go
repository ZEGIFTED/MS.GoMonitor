package internal

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
)

type AgentRepository struct {
}

// AgentThresholds returns Agent Threshold Information
type AgentThresholds struct{}

func (a *AgentRepository) ValidateAgentURL(AgentAPIBaseURL, endpoint string) (string, error) {
	// Parse the AgentAPIBaseURL
	parsedURL, err := url.Parse(AgentAPIBaseURL)
	if err != nil {
		return "", fmt.Errorf("invalid Agent Base URL")
	}

	//if parsedURL.Hostname() == AgentAPIBaseURL {
	return parsedURL.ResolveReference(&url.URL{Path: endpoint}).String(), nil
	//}

	//if host == "" || port == "" || endpoint == "" {
	//	return "", fmt.Errorf("invalid Agent URL")
	//}
	//
	//if protocol != "https" && protocol != "http" {
	//	log.Println("invalid agent protocol in configuration... Using default")
	//
	//	protocol = "http"
	//}
	//
	//agentAddress := fmt.Sprintf("%v://%s:%d/%s", protocol, host, port, endpoint)
	//
	//return agentAddress, nil
}

func (a *AgentRepository) GetAgentThresholds(agentURL string) (AgentThresholds, error) {
	if agentURL == "" {
		return AgentThresholds{}, fmt.Errorf("invalid agent Base URL")
	}

	// Create a custom HTTP client with disabled SSL verification
	httpClient := &http.Client{
		Timeout: constants.HTTPRequestTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := httpClient.Get(agentURL)

	if err != nil {
		return AgentThresholds{}, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return AgentThresholds{}, err
	}

	var apiResponse AgentThresholdResponse
	err = json.Unmarshal(body, &apiResponse)

	log.Println("Agent Threshold API response", apiResponse, err)

	return AgentThresholds{}, nil
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

func (a *AgentRepository) GetAgentServiceStats(agentURL string) ([]ProcessResourceUsage, error) {
	if agentURL == "" {
		return []ProcessResourceUsage{}, fmt.Errorf("invalid agent Base URL")
	}

	// Create a custom HTTP client with disabled SSL verification
	httpClient := &http.Client{
		Timeout: constants.HTTPRequestTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := httpClient.Get(agentURL)

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
