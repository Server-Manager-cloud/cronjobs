package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

type Domain struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Nameserver string `json:"nameserver"`
}

type PocketBaseResponse struct {
	Items []Domain `json:"items"`
}

func getPocketBaseRecords(apiURL string) ([]Domain, error) {
	req, err := http.NewRequest("GET", apiURL+"/api/collections/domains/records", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to retrieve records, HTTP Status: %s", resp.Status)
	}

	var pbResponse PocketBaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&pbResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return pbResponse.Items, nil
}

func updateDomainNameserver(apiURL string, domainID, nameserver string) error {
	payload := map[string]interface{}{
		"nameserver": nameserver,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/api/collections/domains/records/%s", apiURL, domainID), bytes.NewBuffer(payloadBytes))
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update domain %s, HTTP Status: %s", domainID, resp.Status)
	}

	log.Printf("Domain %s updated with nameserver %s", domainID, nameserver)
	return nil
}

func getNameserverFromDNS(domain string) (string, error) {
	cmd := exec.Command("nslookup", "-type=NS", domain)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to execute nslookup: %v", err)
	}

	var nameservers []string
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Print(line)
		if strings.Contains(line, "origin = ns.udag.de") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				nameservers = append(nameservers, "udag")
			}
		}
		if strings.Contains(line, "origin = ns.hetzner.de") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				nameservers = append(nameservers, "hetzner")
			}
		}
	}

	if len(nameservers) == 0 {
		return "unknown", fmt.Errorf("no nameservers found for domain %s", domain)
	}

	return strings.Join(nameservers, ", "), nil
}

func main() {
	apiURL := "https://admin.server-manager.cloud" // replace with your PocketBase instance URL

	// Get domains from PocketBase
	domains, err := getPocketBaseRecords(apiURL)
	if err != nil {
		log.Fatalf("Error retrieving domains: %v", err)
	}

	if len(domains) == 0 {
		log.Println("No domains found in PocketBase.")
		return
	}

	// Loop through domains and update nameserver field
	for _, domain := range domains {
		log.Printf("Processing domain: %s", domain.Name)

		// Get nameserver information from nslookup
		nameserver, err := getNameserverFromDNS(domain.Name)
		if err != nil {
			log.Printf("Error getting nameserver for domain %s: %v", domain.Name, err)
		}

		// Update the domain with the nameserver in PocketBase
		err = updateDomainNameserver(apiURL, domain.ID, nameserver)
		if err != nil {
			log.Printf("Error updating domain %s: %v", domain.Name, err)
			continue
		}
	}

	log.Println("Completed processing all domains.")
}
