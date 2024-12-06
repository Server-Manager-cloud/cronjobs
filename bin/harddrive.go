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

// getDiskUsage calculates the disk usage percentage for a given path
func getDiskUsage(path string) (int, error) {
	// Run the `df` command for the given path
	cmd := exec.Command("df", "-h", path)
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return 0, fmt.Errorf("failed to execute df command: %v", err)
	}

	// Parse the output of `df`
	lines := strings.Split(out.String(), "\n")
	if len(lines) < 2 {
		return 0, fmt.Errorf("unexpected output from df command")
	}

	fields := strings.Fields(lines[1]) // Second line contains the stats
	if len(fields) < 5 {
		return 0, fmt.Errorf("unexpected output format")
	}

	// Extract the usage percentage (e.g., "45%")
	usageStr := strings.TrimSuffix(fields[4], "%")
	usagePercentage, err := strconv.Atoi(usageStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse usage percentage: %v", err)
	}

	return usagePercentage, nil
}

// sendUsageToPocketBase sends the usage data to the PocketBase API
func sendUsageToPocketBase(usage int, path, id, domain string) error {
	// now := time.Now()
	collection := "harddrives"

	// Check if the current minutes value is 0
	// if now.Minute() == 0 {
	// Construct the full API URL using the domain from smc.json
	apiURL := fmt.Sprintf("https://%s/api/collections/%s/records", domain, collection)

	// Create the payload with ID and path
	payload := fmt.Sprintf(`{"usagePercentage": %d, "path": "%s", "server": "%s"}`, usage, path, id)
	payloadBytes := []byte(payload)

	// Create the HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Set headers for the request
	// req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")

	// Execute the request
	client := &http.Client{}

	resp, err := client.Do(req)
	fmt.Println(err)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	fmt.Println(resp)
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to update status, HTTP Status: %s", resp.Status)
	}
	// }

	return nil
}

// getMountedPaths retrieves all mounted paths (excluding special file systems)
func getMountedPaths() ([]string, error) {
	// Run the `df` command to get all mounted file systems
	cmd := exec.Command("df", "-h")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to execute df command: %v", err)
	}

	// Parse the output of `df`
	lines := strings.Split(out.String(), "\n")
	var paths []string

	// Skip the first line (header) and extract paths from the second line onward
	for _, line := range lines[1:] {
		// Split by whitespace to separate the fields
		fields := strings.Fields(line)
		if len(fields) > 0 {
			// Add the path (field[0]) if it looks like a valid file system path
			paths = append(paths, fields[0])
		}
	}

	return paths, nil
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

	// Get the ID from the environment variable
	id := os.Getenv("ID")
	if id == "" {
		log.Fatal("ID environment variable not set.")
	}

	// Get all mounted file system paths
	paths, err := getMountedPaths()
	if err != nil {
		log.Fatalf("Error getting mounted paths: %v", err)
	}

	// Iterate over each path and get disk usage
	for _, path := range paths {
		// Get the disk usage percentage for the current path
		usagePercentage, err := getDiskUsage(path)
		if err != nil {
			log.Printf("Error getting disk usage for %s: %v", path, err)
			continue
		}

		// Send the usage data to PocketBase
		err = sendUsageToPocketBase(usagePercentage, path, id, config.Domain)
		if err != nil {
			log.Printf("Error sending usage for %s to PocketBase: %v", path, err)
			continue
		}

		fmt.Printf("Hard drive usage successfully reported for %s! Current usage: %d%%\n", path, usagePercentage)
	}
}
