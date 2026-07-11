# Contributing

Hey, thanks for checking this out. I'm building this solo in my free time to solve my own rate-limiting headaches. It works well for my current setup, but it's still an early alpha and I definitely can't catch every edge case on my own.

If you want to submit a PR, that would be awesome. Just keep a few things in mind so we don't break the architecture:

* **Keep it sandboxed:** The whole point of this mesh is security. Don't add dependencies or scripts that require root. Rely on the systemd sandbox (`ProtectSystem=strict`, `NoNewPrivileges=true`).
* **Keep it stateless:** Core nodes shouldn't store anything permanently. Everything needs to fit into the Rust backup cycle.
* **Test the concurrency:** If you're touching the Go orchestrator, make sure there are no hanging goroutines or channel locks.

**How to jump in:**
Just fork the repo, use `config.example.json` (please don't commit your actual proxy IPs!), and open a PR. 

If it's a quick fix or optimization, just send it. If you want to do a massive rewrite of the routing logic, open an issue first so we can chat before you spend hours coding something that might not fit.
