Markdown
# Architecture Spec: 9-Node Stateless Redlib Mesh

### 1. Topology
* **Edge (TLS/Cache):** 9x geographically distributed VPS nodes ($3–$15/yr/node). Cloudflare fronting.
* **Core (Compute):** 2x CachyOS hubs (Active/Active). No public DNS. Firewalls restricted to Edge IP ingress.
* **Egress (Proxy Pool):** Core hubs fetch cache-miss data via residential proxy pool.

### 2. Constraints & Budget
* **Annual Spend:** ~$1,536 (Compute) + ~$964 (Proxy bandwidth @ ~$1/GB) = ~$2,500.
* **Throughput:** Requires >95% cache hit ratio (`proxy_cache_use_stale`).
* **Protection:** Nginx `limit_req` zones per IP to drop scrapers (429) before backend traversal.

### 3. Core Sandboxing (Systemd)
```ini
[Service]
ExecStart=/usr/local/bin/redlib
Restart=always
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
StateDirectory=redlib
CapabilityBoundingSet=
SystemCallFilter=@system-service
DeviceAllow=/dev/null r
4. Edge Orchestration (Go)
Concurrent SSH execution via goroutines on admin machine.

5. Stateless Backups (Rust)
Systemd timer-triggered snapshot/AES-256 GPG encryption.


---

### 2. `main.go`
The Go binary for concurrent node orchestration.

```go
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"golang.org/x/crypto/ssh"
)

type Node struct { Name, Host string; Port int; User string }
type Config struct { PrivateKeyPath string; Nodes []Node }

func runRemoteCommand(node Node, signer ssh.Signer, cmd string, wg *sync.WaitGroup) {
	defer wg.Done()
	config := &ssh.ClientConfig{User: node.User, Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)}, HostKeyCallback: ssh.InsecureIgnoreHostKey()}
	client, _ := ssh.Dial("tcp", fmt.Sprintf("%s:%d", node.Host, node.Port), config)
	defer client.Close()
	session, _ := client.NewSession()
	defer session.Close()
	out, _ := session.CombinedOutput(cmd)
	fmt.Printf("[%s]: %s\n", node.Name, string(out))
}

func main() {
	configData, _ := ioutil.ReadFile("config.json")
	var config Config
	json.Unmarshal(configData, &config)
	key, _ := ioutil.ReadFile(config.PrivateKeyPath)
	signer, _ := ssh.ParsePrivateKey(key)
	var wg sync.WaitGroup
	for _, node := range config.Nodes {
		wg.Add(1)
		go runRemoteCommand(node, signer, os.Args[1], &wg)
	}
	wg.Wait()
}
