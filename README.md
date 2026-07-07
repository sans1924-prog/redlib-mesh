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
