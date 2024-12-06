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
	"strings"
)

// Config structure to load `smc.json`
type Config struct {
	Domain string `json:"domain"`
}

// loadConfig loads the domain configuration from `smc.json`
func loadConfig(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open config file: %v", err)
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return "", fmt.Errorf("failed to decode config file: %v", err)
	}

	return config.Domain, nil
}

// loadEnv reads the `.env` file and retrieves the server ID
func loadEnv(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open .env file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			return strings.TrimPrefix(line, "ID="), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading .env file: %v", err)
	}

	return "", fmt.Errorf("ID not found in .env file")
}

// getCertbotCertificates retrieves domain names from certbot
func getCertbotCertificates() ([]string, error) {
	cmd := exec.Command("certbot", "certificates")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to execute certbot command: %v", err)
	}

	var certificates []string
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Certificate Name:") {
			certName := strings.Replace(line, "Certificate Name:", "", 1)
			certificates = append(certificates, certName)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading certbot output: %v", err)
	}

	return certificates, nil
}

// checkDomainExists checks if a domain already exists in PocketBase
func checkDomainExists(apiURL, domain, serverID string) (string, error) {
	// Query PocketBase to check if the domain already exists
	req, err := http.NewRequest("GET", fmt.Sprintf("%s?filter=name=%s", apiURL, domain), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var records []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&records); err != nil {
			return "", fmt.Errorf("failed to decode response: %v", err)
		}

		// If records found, return the first record's ID (to update it)
		if len(records) > 0 {
			if recordID, ok := records[0]["id"].(string); ok {
				return recordID, nil
			}
		}
	}

	return "", nil // Return empty if no existing domain found
}

// sendDomainsToPocketBase sends the domain names to the PocketBase API
func sendDomainsToPocketBase(domain string, domains []string, serverID string) error {
	apiURL := fmt.Sprintf("https://%s/api/collections/domains/records", domain)

	for _, certDomain := range domains {
		// Get the DNS provider for the domain
		dnsProvider := "unkown"

		// Check if the domain already exists
		recordID, err := checkDomainExists(apiURL, certDomain, serverID)
		if err != nil {
			return fmt.Errorf("failed to check if domain exists: %v", err)
		}

		// Prepare payload with the DNS provider info
		payload := map[string]interface{}{
			"server":     serverID,
			"name":       certDomain,
			"nameserver": dnsProvider,
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %v", err)
		}

		var req *http.Request
		if recordID != "" {
			// Update existing entry
			req, err = http.NewRequest("PUT", fmt.Sprintf("%s/%s", apiURL, recordID), bytes.NewBuffer(payloadBytes))
		} else {
			// Create new entry
			req, err = http.NewRequest("POST", apiURL, bytes.NewBuffer(payloadBytes))
		}
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("failed to send domain %s, HTTP Status: %s", certDomain, resp.Status)
		}

		log.Printf("Domain %s successfully processed with DNS provider %s.", certDomain, dnsProvider)
	}

	return nil
}

func main() {
	// Load configuration from `smc.json`
	configDomain, err := loadConfig("smc.json")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Load server ID from `.env`
	serverID, err := loadEnv(".env")
	if err != nil {
		log.Fatalf("Error loading environment: %v", err)
	}

	// Get certificate domains
	domains, _ := getCertbotCertificates()

	if len(domains) == 0 {
		log.Println("No certificates found.")
		return
	}

	// Send domains to PocketBase
	err = sendDomainsToPocketBase(configDomain, domains, serverID)
	if err != nil {
		log.Fatalf("Error sending domains to PocketBase: %v", err)
	}

	fmt.Println("All domains successfully processed.")
}
