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
			certName := strings.TrimSpace(strings.TrimPrefix(line, "Certificate Name:"))
			certificates = append(certificates, certName)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading certbot output: %v", err)
	}

	return certificates, nil
}

// sendDomainsToPocketBase sends the domain names to the PocketBase API
func sendDomainsToPocketBase(domain string, domains []string, serverID string) error {
	apiURL := fmt.Sprintf("https://%s/api/collections/domains/records", domain)

	for _, certDomain := range domains {
		payload := map[string]interface{}{
			"server": serverID,
			"name":   certDomain,
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %v", err)
		}

		req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payloadBytes))
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

		log.Printf("Domain %s successfully sent to PocketBase.", certDomain)
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
	domains, err := getCertbotCertificates()
	if err != nil {
		log.Fatalf("Error getting certbot certificates: %v", err)
	}

	if len(domains) == 0 {
		log.Println("No certificates found.")
		return
	}

	// Send domains to PocketBase
	err = sendDomainsToPocketBase(configDomain, domains, serverID)
	if err != nil {
		log.Fatalf("Error sending domains to PocketBase: %v", err)
	}

	fmt.Println("All domains successfully reported to PocketBase.")
}
