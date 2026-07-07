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

*See `main.go` in the repository.*

## 5. Stateless Backups (Rust)

Systemd timer-triggered snapshot/AES-256 GPG encryption.

*See `main.rs` in the repository.*

## Deployment

1. Configure `config.json` with your edge node IPs.
2. Compile Go orchestrator: `go build -ldflags="-s -w" -o redlib-mesh main.go`.
3. Deploy configuration and systemd unit files to core hubs.
4. Set up systemd timer on core hubs to trigger main.rs for encrypted backups.
4. Deploy configuration and systemd unit files to core hubs.
5. Set up systemd timer on core hubs to trigger `main.rs` for encrypted backups.
