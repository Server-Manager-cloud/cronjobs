package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Load the domain configuration from the smc.json file
type Config struct {
	Domain string `json:"domain"`
}

// loadEnv reads the .env file and loads its contents into environment variables.
func loadEnv(filePath string) error {
	// Open the .env file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open .env file: %v", err)
	}
	defer file.Close()

	// Read each line from the .env file
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines or comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split line into key and value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // Skip lines that don't have key-value pairs
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Set the environment variable
		err := os.Setenv(key, value)
		if err != nil {
			return fmt.Errorf("failed to set environment variable: %v", err)
		}
	}

	// Check for any errors during scanning
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading .env file: %v", err)
	}

	return nil
}

// getCPUUsage retrieves the current CPU usage as a percentage.
func getCPUUsage() (float64, error) {
	// Run the `top` command to get CPU usage
	cmd := exec.Command("top", "-bn1", "|", "grep", "'%Cpu'", "|", "sed", "'s/.*, *\\([0-9.]*\\)%* id.*/\\1/'", "|", "awk", "'{print 100 - $1}'")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return 0, fmt.Errorf("failed to get CPU usage: %v", err)
	}

	// Parse the output of `top` to extract CPU usage percentage
	usageStr := strings.TrimSpace(out.String())
	usage, err := strconv.ParseFloat(usageStr, 64) // Use strconv.ParseFloat instead of fmt.ParseFloat
	if err != nil {
		return 0, fmt.Errorf("failed to parse CPU usage: %v", err)
	}

	return usage, nil
}

// sendUsageToPocketBase sends the CPU usage data to the PocketBase API
func sendUsageToPocketBase(cpuUsage float64, id, domain string) error {
	now := time.Now()
	collection := "cpu"

	// Check if the current minutes value is 0
	if now.Minute() == 0 {
		// Construct the full API URL using the domain from smc.json
		apiURL := fmt.Sprintf("https://%s/api/collections/%s/records", domain, collection)
		apiToken := os.Getenv("API_TOKEN") // Use API_TOKEN from .env

		// Create the payload with ID and CPU usage
		payload := fmt.Sprintf(`{"cpuUsage": %.2f, "id": "%s"}`, cpuUsage, id)
		payloadBytes := []byte(payload)

		// Create the HTTP request
		req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payloadBytes))
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %v", err)
		}

		// Set headers for the request
		req.Header.Set("Authorization", "Bearer "+apiToken)
		req.Header.Set("Content-Type", "application/json")

		// Execute the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// Check the response status
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("failed to update status, HTTP Status: %s", resp.Status)
		}
	}

	return nil
}

// loadConfig reads the smc.json configuration file and loads the domain.
func loadConfig(filePath string) (*Config, error) {
	// Open the smc.json file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open smc.json file: %v", err)
	}
	defer file.Close()

	// Parse the JSON content
	var config Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse smc.json: %v", err)
	}

	return &config, nil
}

func main() {
	// Load environment variables from the .env file
	err := loadEnv(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Load configuration from smc.json
	config, err := loadConfig("smc.json")
	if err != nil {
		log.Fatalf("Error loading smc.json file: %v", err)
	}

	// Get the server ID from the environment variable
	id := os.Getenv("ID")
	if id == "" {
		log.Fatal("ID environment variable not set.")
	}

	// Get the CPU usage
	cpuUsage, err := getCPUUsage()
	if err != nil {
		log.Fatalf("Error getting CPU usage: %v", err)
	}

	// Send the CPU usage data to PocketBase
	err = sendUsageToPocketBase(cpuUsage, id, config.Domain)
	if err != nil {
		log.Fatalf("Error sending CPU usage to PocketBase: %v", err)
	}

	fmt.Printf("CPU usage successfully reported! Current usage: %.2f%%\n", cpuUsage)
}
