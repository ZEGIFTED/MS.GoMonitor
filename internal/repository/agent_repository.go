package repository

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
	mstypes "github.com/ZEGIFTED/MS.GoMonitor/types"
)

type AgentRepository struct{}

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

func (a *AgentRepository) GetAgentThresholds(agentURL string) (mstypes.AgentThresholdResponse, error) {
	if agentURL == "" {
		return mstypes.AgentThresholdResponse{}, fmt.Errorf("invalid agent Base URL")
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
		return mstypes.AgentThresholdResponse{}, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return mstypes.AgentThresholdResponse{}, err
	}

	var apiResponse mstypes.AgentThresholdResponse
	if err_ := json.Unmarshal(body, &apiResponse); err_ != nil {
		return mstypes.AgentThresholdResponse{}, err
	}

	log.Println("Agent Threshold API response", apiResponse)

	return apiResponse, nil
}

func ServerResourceDetails(baseURL string, limit int) ([]mstypes.ProcessResourceUsage, error) {
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
	var processes []mstypes.ProcessResourceUsage
	if err := json.Unmarshal(body, &processes); err != nil {
		return nil, err
	}

	return processes, nil
}

func (a *AgentRepository) GetAgentServiceStats(agentURL string) (mstypes.ProcessResponse, error) {
	if agentURL == "" {
		return nil, fmt.Errorf("invalid agent Base URL")
	}

	// Create a custom HTTP client with disabled SSL verification
	// httpClient := &http.Client{
	// 	Timeout: constants.HTTPRequestTimeout,
	// 	Transport: &http.Transport{
	// 		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	// 	},
	// }
	log.Println("Retrieving Device Processes")
	resp, err := http.Get(agentURL)

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
	var processes mstypes.ProcessResponse
	if err := json.Unmarshal([]byte(body), &processes); err != nil {
		log.Println("Error unmarshalling JSON:", err)
		return nil, err
	}

	return processes, nil
}
