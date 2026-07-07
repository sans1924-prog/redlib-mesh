package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
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
func runRemoteCommand(node Node, signer ssh.Signer, cmd string) error {
	config := &ssh.ClientConfig{
		User: node.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		// Note: For strict production, replace InsecureIgnoreHostKey with a KnownHosts callback
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second, // Prevents hanging on dead nodes
	}

	address := fmt.Sprintf("%s:%d", node.Host, node.Port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
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

	// 4. Execute Concurrently with Error Tracking
	var wg sync.WaitGroup
	errorChan := make(chan error, len(config.Nodes))

	fmt.Printf("Executing on %d nodes...\n\n", len(config.Nodes))

	for _, node := range config.Nodes {
		wg.Add(1)
		go func(n Node) {
			defer wg.Done()
			err := runRemoteCommand(n, signer, cmd)
			if err != nil {
				// Send to forensic log
				log.Printf("[FAILED] %s: %v\n", n.Name, err)
				// Send to console channel
				errorChan <- fmt.Errorf("[\033[31mFAIL\033[0m] %s: %v", n.Name, err)
			}
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
		fmt.Printf("\nCompleted with errors. %d/%d nodes failed. Check mesh-forensic.log for details.\n", failCount, len(config.Nodes))
		os.Exit(1)
	}

	fmt.Println("\nAll nodes updated successfully.")
}
