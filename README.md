# Redlib Stateless Mesh

[![Go CI/CD Pipeline](https://github.com/sans1924-prog/redlib-mesh/actions/workflows/ci.yml/badge.svg)](https://github.com/sans1924-prog/redlib-mesh/actions)
   
> **Caution:** **Project Status: Early Alpha & Experimental**
> 
> Redlib Stateless Mesh is an experimental framework currently in an early alpha stage of development. While the core topology, systemd sandboxing, and concurrent orchestration logic are functional and production-verified, the overall pipeline is still actively evolving.
> * **Expect Changes:** Structural refactors, configuration adjustments, and optimization updates should be anticipated as real-world testing continues.
> * **Operational Warning:** Deploy with care in critical environments and closely monitor your rotating proxy bandwidth consumption during initial setup.

---

An architecture specification for a high-traffic Redlib instance designed to bypass upstream IP bans and rate-limiting while maintaining a strict, cost-effective budget footprint.

## 1. Topology

* **Edge (TLS/Cache):** 9x geographically distributed VPS nodes ($3–$15/yr/node) fronted by Cloudflare.
* **Core (Compute):** 2x CachyOS hubs (Active/Active). No public DNS. Firewalls restricted to Edge IP ingress only.
* **Egress (Proxy Pool):** Core hubs fetch cache-miss data via rotating residential proxy pool to mimic consumer traffic.

## 2. Constraints & Budget

* **Annual Spend:** ~$1,536 (Compute) + ~$964 (Proxy bandwidth @ ~$1/GB) = ~$2,500 total in full scale.
* **Throughput:** Requires >95% cache hit ratio using Nginx `proxy_cache_use_stale`.
* **Protection:** Edge nodes utilize strict `limit_req` zones per IP to drop scrapers (429) before backend traversal.

## 3. Core Sandboxing (Systemd)

Core nodes run bare-metal CachyOS with `sudo` replaced by `opendoas`.

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

## 4. Edge Orchestration (Go Control Plane)

A local-run management binary (`main.go`) written in Go for highly resilient, concurrent SSH configuration deployment via managed goroutines.

* **Semaphore Concurrency Control:** Enforces a strict limit of 10 concurrent active SSH connection channels to prevent local file descriptor exhaustion and hub IP throttling.
* **Automated Fault Mitigation:** Features a 3-tier internal retry loop with exponential backoff (`2s -> 4s -> 8s`) to cleanly absorb transient network degradation on low-cost edge nodes.
* **Graceful Interrupt Handling:** Traps `SIGINT` / `SIGTERM` (Ctrl+C) via context propagation to safely close active TCP socket blocks without leaving zombie execution frames or orphaned edge connections.
* **Two-Tier Forensic Logging:** Keeps terminal outputs clean by swallowing transient retry warnings while piping deep stack execution traces to a local `mesh-forensic.log` file.

## 5. Stateless Backups (Rust)

A systemd timer-triggered utility (`main.rs`) written in Rust for dynamic snapshot isolation and AES-256 GPG data durability.

---

## Local Development & Zero-Budget Testing

You do not need an enterprise budget or active remote nodes to test or contribute to this project. The architecture is explicitly designed to scale down to a single local machine for development:

1. **Mock Environment:** Populate `config.json` with local loopback endpoints or dead target ports (e.g., `127.0.0.1:9999`) to safely test the orchestrator's concurrency tracking, backoff retries, and error routing frameworks.
2. **Free Ingress Isolation:** For functional sandbox deployments, utilize free **Cloudflare Tunnels** (`cloudflared`) to manage incoming edge configurations securely without opening inbound ports on your host router.
3. **Free IP Rotation:** Route upstream core traffic through a local **Tor daemon SOCKS5 proxy** (`socks5://127.0.0.1:9050`) to evaluate scraper defenses and rate-limit handling for zero external cost.

## Deployment

1. Configure `config.json` based on the template in `config.example.json`.
2. Compile the Go orchestrator control plane:
   ```bash
   go build -ldflags="-s -w" -o redlib-mesh main.go
   ```
3. Deploy configurations and systemd unit files to core hubs.
4. Set up the systemd timer architecture on your compute tier to handle automated cryptographic backups via `main.rs`.

## Contributing

I am developing this project solo, and pull requests or architectural reviews are highly welcome. If you have experience with Nginx proxy-cache tuning, Go systems programming, Linux sandboxing, or Rust system tools, feel free to jump in.

Check the **Issues** tab for tasks explicitly tagged `good first issue` and `help wanted`, or read `CONTRIBUTING.md` for our sandboxing standard guidelines.
