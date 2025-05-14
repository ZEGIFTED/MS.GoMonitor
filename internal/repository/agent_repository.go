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

func (a *AgentRepository) ValidateAgentURL(AgentAPIBaseURL, endpoint string) (*http.Client, string, error) {
	// Parse the AgentAPIBaseURL
	parsedURL, err := url.Parse(AgentAPIBaseURL)
	if err != nil {
		return nil, "", fmt.Errorf("invalid Agent Base URL")
	}

	// Create a custom HTTP client with disabled SSL verification
	httpClient := &http.Client{
		Timeout: constants.HTTPRequestTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	return httpClient, parsedURL.ResolveReference(&url.URL{Path: endpoint}).String(), nil
}

func (a *AgentRepository) GetAgentThresholds(agentHttpClient *http.Client, agentURL string) (mstypes.AgentThresholdResponse, error) {
	if agentURL == "" {
		return mstypes.AgentThresholdResponse{}, fmt.Errorf("invalid agent Base URL")
	}

	resp, err := agentHttpClient.Get(agentURL)

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
	route := baseURL + "/api/v1/agent/resource-usage?limit=" + strconv.Itoa(limit)
	resp, err := http.Get(route)

	if err != nil {
		return nil, fmt.Errorf("route: %s. Error >>> %s", route, err.Error())
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

func (a *AgentRepository) GetAgentServiceStats(agentHttpClient *http.Client, agentURL string) (mstypes.ProcessResponse, error) {
	if agentURL == "" {
		return nil, fmt.Errorf("invalid agent Base URL")
	}

	log.Println("Retrieving Device Processes")
	resp, err := agentHttpClient.Get(agentURL)

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

func (a *AgentRepository) GetAgentContainerStats(agentHttpClient *http.Client, agentURL string) (mstypes.AgentContainerResponse, error) {
	if agentURL == "" {
		return mstypes.AgentContainerResponse{}, fmt.Errorf("invalid agent Base URL")
	}

	resp, err := agentHttpClient.Get(agentURL)

	if err != nil {
		return mstypes.AgentContainerResponse{}, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return mstypes.AgentContainerResponse{}, err
	}

	var apiResponse mstypes.AgentContainerResponse
	if err_ := json.Unmarshal(body, &apiResponse); err_ != nil {
		return mstypes.AgentContainerResponse{}, err
	}

	log.Println("Agent Threshold API response", apiResponse)

	return apiResponse, nil
}
