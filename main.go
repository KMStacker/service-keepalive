package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	fmt.Println("Starting service-keepalive worker...")
	loadEnv(".env")

	// RENDER SITE KEEPALIVE
	renderURL := os.Getenv("RENDER_SITE_URL")
	if renderURL != "" {
		err := pingEndpoint(renderURL)
		if err != nil {
			fmt.Printf("Render ping failed: %v\n", err)
		}
	} else {
		fmt.Println("RENDER_SITE_URL not set, skipping Render ping.")
	}

	// AIVEN SERVICE KEEPALIVE
	aivenToken := os.Getenv("AIVEN_API_TOKEN")
	aivenProject := os.Getenv("AIVEN_PROJECT_NAME")
	aivenService := os.Getenv("AIVEN_SERVICE_NAME")

	if aivenToken == "" || aivenProject == "" || aivenService == "" {
		fmt.Println("Error: AIVEN_API_TOKEN, AIVEN_PROJECT_NAME, or AIVEN_SERVICE_NAME is not set.")
		os.Exit(1)
	}

	err := keepAivenAlive(aivenToken, aivenProject, aivenService)
	if err != nil {
		fmt.Printf("Aiven keepalive failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Worker finished all checks successfully.")
	os.Exit(0)
}

func pingEndpoint(url string) error {
	client := &http.Client{
		Timeout: 99 * time.Second,
	}

	fmt.Printf("Sending HTTP GET request to: %s\n", url)
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("network request failed: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Received response status: %s\n", resp.Status)

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("received non-success status code: %d", resp.StatusCode)
	}

	return nil
}

type AivenServiceResponse struct {
	Service struct {
		State string `json:"state"`
	} `json:"service"`
}

type AivenPowerPayload struct {
	Powered bool `json:"powered"`
}

func keepAivenAlive(apiToken, project, service string) error {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	url := fmt.Sprintf("https://api.aiven.io/v1/project/%s/service/%s", project, service)
	fmt.Printf("Checking Aiven service state from: %s\n", url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Aiven API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var serviceData AivenServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&serviceData); err != nil {
		return fmt.Errorf("failed to parse Aiven response: %w", err)
	}

	currentState := serviceData.Service.State
	fmt.Printf("Aiven service current state: %s\n", currentState)

	if currentState == "POWERED_OFF" {
		fmt.Println("Service is POWERED_OFF. Sending power-on request...")
		return powerOnAivenService(client, apiToken, url)
	}

	fmt.Println("Aiven service is active.")
	return nil
}

func powerOnAivenService(client *http.Client, apiToken, url string) error {
	payload := AivenPowerPayload{Powered: true}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal power payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create power-on request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute power-on request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Aiven power-on returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	fmt.Println("Aiven service power-on request sent successfully.")
	return nil
}

func loadEnv(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
		}
	}
}