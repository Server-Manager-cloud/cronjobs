package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
)

func main() {
	// List of Go script paths to run
	scripts := []string{
		"bin/harddrive.go",
		"bin/cpu.go",
		"bin/domains.go",
		"bin/nameserver.go",
	}

	// Iterate through the scripts and run each
	for _, scriptPath := range scripts {
		fmt.Printf("Running script: %s\n", scriptPath)

		// Command to execute the script
		cmd := exec.Command("go", "run", scriptPath)

		// Capture the script's output
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		// Run the command
		err := cmd.Run()
		if err != nil {
			log.Printf("Error running script %s: %v\nStderr: %s\n", scriptPath, err, stderr.String())
			continue
		}

		// Print the script's output
		fmt.Printf("Output from %s:\n%s\n", scriptPath, stdout.String())
	}

	fmt.Println("All scripts executed.")
}
