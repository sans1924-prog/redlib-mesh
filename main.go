package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
)

type Node struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Port int    `json:"port"`
	User string `json:"user"`
}

type Config struct {
	PrivateKeyPath string `json:"private_key_path"`
	Nodes          []Node `json:"nodes"`
}

// initLogger sets up forensic logging to a local file
func initLogger() *os.File {
	logFile, err := os.OpenFile("mesh-forensic.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Warning: Could not open forensic log file: %v\n", err)
		return nil
	}
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	return logFile
}

// runRemoteCommand connects via SSH and executes the command, returning an error if it fails
func runRemoteCommand(ctx context.Context, node Node, signer ssh.Signer, cmd string) error {
	config := &ssh.ClientConfig{
		User: node.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		// SECURITY TODO: For strict production, replace InsecureIgnoreHostKey with a KnownHosts callback
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         8 * time.Second, // Tighter timeout to fail-fast during network degradation
	}

	address := fmt.Sprintf("%s:%d", node.Host, node.Port)
	
	// Create a channel to handle the SSH dialing concurrently with context cancellation
	dialResult := make(chan struct {
		client *ssh.Client
		err    error
	}, 1)

	go func() {
		client, err := ssh.Dial("tcp", address, config)
		dialResult <- struct {
			client *ssh.Client
			err    error
		}{client, err}
	}()

	var client *ssh.Client
	select {
	case <-ctx.Done():
		return fmt.Errorf("operation cancelled by user")
	case res := <-dialResult:
		if res.err != nil {
			return fmt.Errorf("dial failed: %w", res.err)
		}
		client = res.client
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("session failed: %w", err)
	}
	defer session.Close()

	out, err := session.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("execution failed (%v). Output: %s", err, string(out))
	}

	fmt.Printf("[\033[32mOK\033[0m] %s: %s", node.Name, string(out))
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: redlib-mesh \"<command to run>\"")
		os.Exit(1)
	}
	cmd := os.Args[1]

	// 0. Setup Graceful Shutdown Context (Catches Ctrl+C)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\n[\033[33m!\033[0m] Interrupt received. Gracefully shutting down connections...")
		cancel()
	}()

	// 1. Setup Forensic Logging
	logFile := initLogger()
	if logFile != nil {
		defer logFile.Close()
	}
	log.Printf("--- Starting Execution: %s ---", cmd)

	// 2. Load Configuration
	configData, err := os.ReadFile("config.json")
	if err != nil {
		fmt.Printf("Fatal: Cannot read config.json: %v\n", err)
		os.Exit(1)
	}

	var config Config
	if err := json.Unmarshal(configData, &config); err != nil {
		fmt.Printf("Fatal: Cannot parse config.json: %v\n", err)
		os.Exit(1)
	}

	// 3. Load SSH Key
	key, err := os.ReadFile(config.PrivateKeyPath)
	if err != nil {
		fmt.Printf("Fatal: Cannot read SSH key at %s: %v\n", config.PrivateKeyPath, err)
		os.Exit(1)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		fmt.Printf("Fatal: Cannot parse SSH key: %v\n", err)
		os.Exit(1)
	}

	// 4. Execute Concurrently with Error Tracking, Retries, and Rate Limiting
	var wg sync.WaitGroup
	errorChan := make(chan error, len(config.Nodes))
	
	// SEMAPHORE: Limit to exactly 10 concurrent SSH connections at once
	maxConcurrentConnections := 10
	semaphore := make(chan struct{}, maxConcurrentConnections)

	fmt.Printf("Executing on %d nodes (Max Concurrency: %d). Press Ctrl+C to abort...\n\n", len(config.Nodes), maxConcurrentConnections)

	for _, node := range config.Nodes {
		wg.Add(1)
		go func(n Node) {
			defer wg.Done()

			// Acquire a semaphore slot before starting
			semaphore <- struct{}{} 
			defer func() { <-semaphore }() // Release slot when done

			const maxRetries = 3
			var lastErr error
			backoff := 2 * time.Second

			for attempt := 1; attempt <= maxRetries; attempt++ {
				// Check if the user hit Ctrl+C before starting the next attempt
				if ctx.Err() != nil {
					errorChan <- fmt.Errorf("[\033[33mABORTED\033[0m] %s: Cancelled by user", n.Name)
					return
				}

				lastErr = runRemoteCommand(ctx, n, signer, cmd)
				if lastErr == nil {
					return // Success
				}

				// If it's a cancellation error, don't retry, just exit
				if ctx.Err() != nil {
					errorChan <- fmt.Errorf("[\033[33mABORTED\033[0m] %s: Cancelled during execution", n.Name)
					return
				}

				log.Printf("[RETRY] %s (Attempt %d/%d) failed: %v. Retrying in %v...\n", n.Name, attempt, maxRetries, lastErr, backoff)

				if attempt < maxRetries {
					// Wait for backoff OR until user hits Ctrl+C
					select {
					case <-time.After(backoff):
						backoff *= 2
					case <-ctx.Done():
						errorChan <- fmt.Errorf("[\033[33mABORTED\033[0m] %s: Cancelled during backoff", n.Name)
						return
					}
				}
			}

			log.Printf("[FATAL CRITICAL] %s execution exhausted after %d attempts: %v\n", n.Name, maxRetries, lastErr)
			errorChan <- fmt.Errorf("[\033[31mFAIL\033[0m] %s (Exhausted %d Retries): %v", n.Name, maxRetries, lastErr)
		}(node)
	}

	wg.Wait()
	close(errorChan)

	// 5. Summarize Results
	failCount := 0
	fmt.Println("\n--- Execution Summary ---")
	for err := range errorChan {
		fmt.Println(err)
		failCount++
	}

	if failCount > 0 {
		fmt.Printf("\nDeployment finished with warnings. %d/%d nodes failed or aborted. Check mesh-forensic.log.\n", failCount, len(config.Nodes))
		os.Exit(1)
	}

	fmt.Println("\nAll nodes updated and verified successfully.")
}
