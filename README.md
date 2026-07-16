# Redlib Stateless Mesh

[![Go CI/CD Pipeline](https://github.com/sans1924-prog/redlib-mesh/actions/workflows/ci.yml/badge.svg)](https://github.com/sans1924-prog/redlib-mesh/actions)
   
> **Caution:** **Project Status: Early Alpha & Experimental**
> 
> Redlib Stateless Mesh is an experimental framework currently in an early alpha stage of development. While the core topology, systemd sandboxing, and concurrent orchestration logic are functional and production-verified, the overall pipeline is still actively evolving.
> * **Expect Changes:** Structural refactors, configuration adjustments, and optimization updates should be anticipated as real-world testing continues.
> * **Operational Warning:** Deploy with care in critical environments and closely monitor your rotating proxy bandwidth consumption during initial setup.

---

An architecture specification and deployment orchestrator for a high-traffic Redlib instance. Designed to bypass upstream IP bans and rate-limiting while maintaining a strict, cost-effective budget footprint.

## ✨ Core Features
* **Stateless Edge:** Edge nodes hold zero configuration or state, acting purely as TLS terminators and cache layers.
* **Concurrent Orchestration:** Custom Go control plane deploys configurations across the mesh using semaphore-throttled goroutines.
* **Automated Resilience:** Built-in exponential backoff loops absorb transient network drops on low-cost edge nodes.
* **Cryptographic Durability:** Rust-based timer utilities ensure core data is snapshotted, AES-256 encrypted, and exported securely.
* **Zero-Trust Compute:** Core hubs operate under strict systemd sandboxing with `NoNewPrivileges` and read-only file systems.

---

## 🏗 Architecture & Topology

### 1. The Network Tier
* **Edge (TLS/Cache):** 9x geographically distributed VPS nodes ($3–$15/yr/node) fronted by Cloudflare. 
* **Core (Compute):** 2x CachyOS hubs (Active/Active). No public DNS. Firewalls restricted to Edge IP ingress only.
* **Egress (Proxy Pool):** Core hubs fetch cache-miss data via rotating residential proxy pool to mimic consumer traffic.

### 2. Constraints & Budget
* **Annual Spend:** ~$1,536 (Compute) + ~$964 (Proxy bandwidth @ ~$1/GB) = ~$2,500 total in full scale.
* **Throughput:** Requires >95% cache hit ratio using Nginx `proxy_cache_use_stale`.
* **Protection:** Edge nodes utilize strict `limit_req` zones per IP to drop scrapers (HTTP 429) before backend traversal.

### 3. Core Sandboxing (Systemd)
Core nodes run bare-metal CachyOS with `sudo` replaced by `opendoas` to minimize attack surfaces.

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
```

### 4. Edge Orchestration (Go Control Plane)
A local-run management binary (`main.go`) written in Go for highly resilient, concurrent SSH configuration deployment.
* **Semaphore Concurrency:** Enforces a strict limit of 10 concurrent active SSH connection channels to prevent local file descriptor exhaustion.
* **Automated Fault Mitigation:** Features a 3-tier internal retry loop with exponential backoff (`2s -> 4s -> 8s`).
* **Graceful Interrupt Handling:** Traps `SIGINT`/`SIGTERM` via context propagation to safely close active TCP sockets.
* **Forensic Logging:** Keeps terminal outputs clean while piping deep stack execution traces to `mesh-forensic.log`.

---

## 💻 Local Development & Zero-Budget Testing

You do not need an enterprise budget or active remote nodes to test or contribute to this project. The architecture is explicitly designed to scale down to a single local machine for development:

1. **Mock Environment:** Populate `config.json` with local loopback endpoints or dead target ports (e.g., `127.0.0.1:9999`). Run `./redlib-mesh "uptime"` to safely test the orchestrator's concurrency tracking, backoff retries, and error routing frameworks.
2. **Free Ingress Isolation:** For functional sandbox deployments, utilize free **Cloudflare Tunnels** (`cloudflared`) to manage incoming edge configurations securely without opening inbound ports on your host router.
3. **Free IP Rotation:** Route upstream core traffic through a local **Tor daemon SOCKS5 proxy** (`socks5://127.0.0.1:9050`) to evaluate scraper defenses and rate-limit handling for zero external cost.

---

## 🚀 Quickstart & Deployment

### Prerequisites
* Go 1.21+ (For orchestrator compilation)
* Rust/Cargo (For backup utility compilation)
* `systemd` target environments (Linux)

### Execution
1. Clone the repository and configure your mesh based on the template:
   ```bash
   cp config.example.json config.json
   ```
2. Compile the Go orchestrator control plane:
   ```bash
   go build -ldflags="-s -w" -o redlib-mesh main.go
   ```
3. Deploy configurations and systemd unit files to core hubs using the binary.
4. Set up the systemd timer architecture on your compute tier to handle automated cryptographic backups via `main.rs`.

---

## 🤝 Contributing

I am developing this project solo, and pull requests or architectural reviews are highly welcome. If you have experience with Nginx proxy-cache tuning, Go systems programming, Linux sandboxing, or Rust system tools, feel free to jump in.

Check the **Issues** tab for tasks explicitly tagged `good first issue` and `help wanted`, or read `CONTRIBUTING.md` for our sandboxing standard guidelines.
