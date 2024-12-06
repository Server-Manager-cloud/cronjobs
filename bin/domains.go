package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
)

const (
	apiURL = "https://admin.server-manager.cloud/api/collections/domains/records"
	apiKey = "YOUR_API_KEY"
)

// Domain represents the structure of a domain record in PocketBase.
type Domain struct {
	Domain string `json:"domain"`
}

func main() {
	domains, err := getCertbotDomains()
	if err != nil {
		log.Fatalf("Error fetching Certbot domains: %v", err)
	}

	for _, domain := range domains {
		err = postToPocketBase(domain)
		if err != nil {
			log.Printf("Error posting domain %s to PocketBase: %v", domain, err)
		} else {
			log.Printf("Successfully posted domain: %s", domain)
		}
	}
}

// getCertbotDomains fetches the list of domains managed by Certbot.
func getCertbotDomains() ([]string, error) {
	cmd := exec.Command("sudo", "certbot", "certificates", "--cert-name", "all")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute Certbot command: %w", err)
	}

	return parseDomains(string(output)), nil
}

// parseDomains extracts domains from the Certbot certificates command output.
func parseDomains(certbotOutput string) []string {
	var domains []string
	lines := bytes.Split([]byte(certbotOutput), []byte("\n"))

	for _, line := range lines {
		if bytes.HasPrefix(line, []byte("  Domains: ")) {
			domainLine := bytes.TrimPrefix(line, []byte("  Domains: "))
			domains = append(domains, string(domainLine))
		}
	}
	return domains
}

// postToPocketBase posts a domain to the PocketBase API.
func postToPocketBase(domain string) error {
	domainData := Domain{Domain: domain}
	jsonData, err := json.Marshal(domainData)
	if err != nil {
		return fmt.Errorf("failed to marshal domain data: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	// req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
