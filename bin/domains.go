package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// getCertbotCertificates runs the certbot command and filters the certificate names.
func getCertbotCertificates() ([]string, error) {
	// Run the `certbot certificates` command
	cmd := exec.Command("certbot", "certificates")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to execute certbot command: %v", err)
	}

	// Parse the output to find lines starting with "Certificate Name:"
	var certificates []string
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Certificate Name:") {
			// Extract the certificate name
			certName := strings.TrimSpace(strings.TrimPrefix(line, "Certificate Name:"))
			certificates = append(certificates, certName)
		}
	}

	// Check for scanning errors
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading certbot output: %v", err)
	}

	return certificates, nil
}

func main() {
	// Get the list of certificate names from certbot
	certificates, err := getCertbotCertificates()
	if err != nil {
		log.Fatalf("Error getting certificates: %v", err)
	}

	// Print the certificate names
	if len(certificates) == 0 {
		fmt.Println("No certificates found.")
		return
	}

	fmt.Println("Certificates managed by certbot:")
	for _, cert := range certificates {
		fmt.Println("- " + cert)
	}
}
