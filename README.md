> [!CAUTION]
> **Project Status: Early Alpha & Experimental**
> 
> **Redlib Stateless Mesh** is an experimental framework currently in an early alpha stage of development. While the core topology, systemd sandboxing, and concurrent orchestration logic are functional and production-verified, the overall pipeline is still actively evolving.
> 
> * **Expect Changes:** Structural refactors, configuration adjustments, and optimization updates should be anticipated as real-world testing continues.
> * **Operational Warning:** Deploy with care in critical environments and closely monitor your rotating proxy bandwidth consumption during initial setup.
> 
> Pull requests, bug reports, and technical architecture feedback are highly welcome!

# Redlib Stateless Mesh

Architecture specification for a high-traffic Redlib instance designed to bypass upstream IP bans and rate-limiting while maintaining a <$2,500/year budget.

## 1. Topology
* **Edge (TLS/Cache):** 9x geographically distributed VPS nodes ($3–$15/yr/node) fronted by Cloudflare.
* **Core (Compute):** 2x CachyOS hubs (Active/Active). No public DNS. Firewalls restricted to Edge IP ingress only.
* **Egress (Proxy Pool):** Core hubs fetch cache-miss data via rotating residential proxy pool to mimic consumer traffic.

## 2. Constraints & Budget
* **Annual Spend:** ~$1,536 (Compute) + ~$964 (Proxy bandwidth @ ~$1/GB) = ~$2,500 total.
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
## 4. Edge Orchestration (Go)
Local-run management binary for concurrent SSH execution via goroutines.

* **Intelligent Error Handling:** Features a two-tier logging system. It provides a concise, real-time summary to the console (with strict network timeouts to prevent hanging on dead nodes) while routing deep-dive execution errors and stack traces to a local `mesh-forensic.log` file.

*See `main.go` in this repository.*

## 5. Stateless Backups (Rust)
Systemd timer-triggered snapshot/AES-256 GPG encryption.

*See `main.rs` in this repository.*

## Deployment
1. Configure `config.json` with your edge node IPs.
2. Compile Go orchestrator: `go build -ldflags="-s -w" -o redlib-mesh main.go`.
3. Deploy configuration and systemd unit files to core hubs.
4. Set up systemd timer on core hubs to trigger `main.rs` for encrypted backups.
