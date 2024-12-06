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
	"runtime"
	"strings"
)

// Struct to represent the payload for PocketBase server_os collection
type ServerOS struct {
	ID      string `json:"server_id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Golang  string `json:"golang"`
	Host    string `json:"host"`
	Kernel  string `json:"kernel"`
	Server  string `json:"server"`
}

// Function to get the current server OS information
func getServerOSInfo() (ServerOS, error) {
	// Retrieve the OS name (e.g., Linux, Windows, Darwin (Mac))
	osName := runtime.GOOS

	// Retrieve the architecture (e.g., amd64, arm64)
	arch := runtime.GOARCH

	// Retrieve the Go version
	goVersion := runtime.Version()

	// Retrieve the host name
	hostname, err := os.Hostname()
	if err != nil {
		return ServerOS{}, fmt.Errorf("failed to get hostname: %v", err)
	}

	// Kernel version (using uname command for Linux)
	var kernelVersion string
	if osName == "linux" {
		cmd := exec.Command("uname", "-r")
		kernelVersionOutput, err := cmd.CombinedOutput()
		if err != nil {
			return ServerOS{}, fmt.Errorf("failed to get kernel version: %v", err)
		}
		kernelVersion = string(kernelVersionOutput)
	} else {
		kernelVersion = "N/A"
	}

	return ServerOS{
		Name:    osName,
		Version: arch,
		Golang:  goVersion,
		Host:    hostname,
		Kernel:  kernelVersion,
	}, nil
}

// Function to load the server ID from the .env file
func loadServerIDFromEnv() (string, error) {
	// Open the .env file
	file, err := os.Open(".env")
	if err != nil {
		return "", fmt.Errorf("failed to open .env file: %v", err)
	}
	defer file.Close()

	var serverID string
	// Read each line of the .env file
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Look for the line starting with ID=
		if strings.HasPrefix(line, "ID=") {
			serverID = strings.TrimPrefix(line, "ID=")
			break
		}
	}

	if serverID == "" {
		return "", fmt.Errorf("server ID not found in .env file")
	}

	return serverID, nil
}

// Function to send server OS info to PocketBase
func sendServerOSToPocketBase(apiURL, serverID string, serverOS ServerOS) error {
	// Add the server ID to the payload
	serverOS.Server = serverID

	// Prepare the payload as JSON
	payloadBytes, err := json.Marshal(serverOS)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Make the HTTP request to PocketBase to insert into the server_os collection
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/collections/server_os/records", apiURL), bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Set content type header
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	fmt.Print(resp)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to insert server OS info, HTTP Status: %s", resp.Status)
	}

	log.Println("Server OS info successfully sent to PocketBase.")
	return nil
}

func main() {
	apiURL := "https://admin.server-manager.cloud" // replace with your PocketBase instance URL

	// Load server ID from .env
	serverID, err := loadServerIDFromEnv()
	if err != nil {
		log.Fatalf("Error loading server ID: %v", err)
	}

	// Get server OS information
	serverOS, err := getServerOSInfo()
	if err != nil {
		log.Fatalf("Error retrieving server OS info: %v", err)
	}

	// Send server OS information to PocketBase
	err = sendServerOSToPocketBase(apiURL, serverID, serverOS)
	if err != nil {
		log.Fatalf("Error sending server OS to PocketBase: %v", err)
	}

	fmt.Println("Server OS info successfully sent to PocketBase.")
}
