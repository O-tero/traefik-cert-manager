                ┌──────────────────────────────────────┐
                │              Traefik                │
                │ ─────────────────────────────────── │
                │ • Reverse Proxy & Load Balancer     │
                │ • Serves services via HTTP/HTTPS    │
                │ • Uses dynamic config for TLS certs │
                └───────────────┬─────────────────────┘
                                │
                ┌───────────────▼─────────────────────┐
                │    SSL/TLS Certificate Manager      │
                │        (Go Application)            │
                │ ─────────────────────────────────── │
                │ • Loads config.yaml (domains, email)│
                │ • Discovers services via Traefik API│
                │ • Calls lego for ACME operations    │
                │ • Stores certs in /certs directory  │
                │ • Updates dynamic_conf.yaml         │
                │ • Triggers Traefik reload           │
                │ • Sends email notifications         │
                └───────────────┬─────────────────────┘
                                │
                ┌───────────────▼─────────────────────┐
                │               lego                  │
                │    (ACME Client Library in Go)      │
                │ ─────────────────────────────────── │
                │ • Registers ACME account (email)    │
                │ • Handles HTTP-01 or DNS-01 challenge│
                │ • Requests certificates from CA      │
                │ • Renews certificates automatically   │
                └───────────────┬─────────────────────┘
                                │
                ┌───────────────▼─────────────────────┐
                │           Let's Encrypt             │
                │   (Certificate Authority via ACME) │
                │ ─────────────────────────────────── │
                │ • Issues SSL/TLS certificates       │
                │ • Validates domain ownership        │
                │ • Provides ACME endpoints           │
                └─────────────────────────────────────┘
