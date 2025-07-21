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


          ┌─────────────────────────────────────────────────────────────────────────┐
          │                             SYSTEM OVERVIEW                             │
          └─────────────────────────────────────────────────────────────────────────┘

                             ┌──────────────────────────────────┐
                             │          Scheduler               │
                             │  (scheduler.go)                 │
                             │----------------------------------│
                             │ • Ticker triggers periodic run  │
                             │ • Context for stop/timeout      │
                             │ • Tracks stats & logs           │
                             └───────────────┬─────────────────┘
                                             │
                       ┌─────────────────────▼─────────────────────┐
                       │          RenewalService                  │
                       │-------------------------------------------│
                       │ • Called by Scheduler on each tick       │
                       │ • Or manual RunOnce() call               │
                       │ • Delegates actual renewal logic         │
                       └───────────────────┬──────────────────────┘
                                           │
                      ┌────────────────────▼──────────────────────┐
                      │        CertificateManager                 │
                      │-------------------------------------------│
                      │ • CheckCertificateHealth()               │
                      │    - Scans certs for expiry              │
                      │ • RenewCertificate(domain)               │
                      │    - Uses ACME client (lego)             │
                      │    - Saves new .crt & .key               │
                      │ • Updates dynamic Traefik config         │
                      └───────────────────┬──────────────────────┘
                                          │
           ┌──────────────────────────────▼─────────────────────────────┐
           │                      ACME (lego client)                    │
           │------------------------------------------------------------│
           │ • Registers ACME account (email, key)                     │
           │ • Handles challenges (HTTP-01 via Traefik or DNS-01)      │
           │ • Contacts Let's Encrypt (CA) for cert issuance/renewal   │
           └──────────────────────────────┬─────────────────────────────┘
                                          │
                     ┌────────────────────▼─────────────────────┐
                     │          Let's Encrypt (CA)             │
                     │------------------------------------------│
                     │ • Validates domain ownership            │
                     │ • Issues new certificates               │
                     │ • Enforces rate limits                  │
                     └──────────────────────────────────────────┘

                                ┌─────────────────────────────────┐
                                │           Traefik              │
                                │--------------------------------│
                                │ • Serves HTTP/HTTPS traffic   │
                                │ • Uses dynamic config file     │
                                │   updated by CertificateManager│
                                │ • Reloads certs automatically │
                                └─────────────────────────────────┘
