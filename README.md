# Automatic SSL/TLS Certificate Manager with Traefik in Go

## Overview

This project demonstrates an **Automatic SSL/TLS Certificate Manager** using Traefik and a Go-based integration. It simplifies the management of SSL/TLS certificates by leveraging Traefik’s ACME (Let's Encrypt) capabilities in a staging environment to avoid rate limits and production errors.

## Features
- Automatic acquisition and renewal of SSL/TLS certificates via Let's Encrypt.
- Integration with Traefik for HTTP-01 and TLS-ALPN-01 challenge solvers.
- Configuration tailored to the Let's Encrypt staging environment for testing.
- Easy-to-follow setup and deployment instructions.

## Requirements
- **Traefik**: Reverse proxy and load balancer.
- **Go**: Minimum version 1.18.
- **DNS Provider**: Configurable to point domains to the server.
- **Linux/Unix environment** (or Docker).
- **Public IP Address** for the server.

## Project Structure
```
.
.
├── cmd                 # Main application entry point
│   └── main.go         # Main function
├── config              # Configuration management
│   └── config.go       # Define and load configurations
├── pkg                 # Core functionality
│   ├── api             # Traefik API integration
│   │   └── api.go      # Handles dynamic certificate updates in Traefik
│   ├── certs           # Certificate management
│   │   ├── request.go  # Request and store certificates
│   │   ├── renew.go    # Renewal logic
│   │   └── storage.go  # Secure certificate storage
│   ├── config          # Shared configuration utilities
│   │   └── loader.go   # Configuration loader
│   ├── notify          # Notification system
│   │   └── notify.go   # Email notifications for expiring certificates
├── web                 # Optional web UI for certificate management
│   └── web.go          # Manage certificate operations through UI
└── README.md           # Documentation

```

## Setup Instructions

### Step 1: Configure DNS Records
1. Access your DNS provider’s control panel.
2. Create A records pointing the domain(s) to your server’s public IP address.
   - Example:
     - Hostname: `staging.myproject.com`
     - Type: `A`
     - Value: `203.0.113.123` (replace with your server IP)
     - TTL: Default or 1 Hour.
3. Use [dnschecker.org](https://dnschecker.org/) to confirm DNS propagation.

---

### Step 2: Set Up Traefik

#### Install Traefik
Use Docker Compose or a binary installation to set up Traefik.

- **Docker Compose**:
  ```bash
  docker-compose up -d
  ```
- **Binary Installation**:
  ```bash
  wget https://github.com/traefik/traefik/releases/download/v2.10.1/traefik_v2.10.1_linux_amd64.tar.gz
  tar -xzvf traefik_v2.10.1_linux_amd64.tar.gz
  sudo mv traefik /usr/local/bin/
  ```

#### Configure Traefik
Create `traefik/traefik.yaml` for static configuration:
```yaml
entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"

certificatesResolvers:
  stagingResolver:
    acme:
      email: "your-email@example.com"
      storage: "acme.json" # Ensure this file has write permissions
      caServer: "https://acme-staging-v02.api.letsencrypt.org/directory"
      httpChallenge:
        entryPoint: "web"
      tlsChallenge: {}

providers:
  file:
    directory: "/etc/traefik/dynamic/"
```

Create `traefik/dynamic_config.yaml` for dynamic configuration:
```yaml
http:
  routers:
    staging-router:
      rule: "Host(`staging.myproject.com`)"
      entryPoints:
        - "websecure"
      tls:
        certResolver: "stagingResolver"

  services:
    staging-service:
      loadBalancer:
        servers:
          - url: "http://127.0.0.1:8080" # Replace with your app’s backend
```

Start Traefik:
```bash
sudo traefik --configFile=/path/to/traefik.yaml
```

---

### Step 3: Configure Go-Based Certificate Management

#### Clone and Set Up the Repository
```bash
git clone https://github.com/your-repo/traefik-cert-manager.git
cd traefik-cert-manager
```

#### Install Dependencies
```bash
go mod tidy
```

#### Run the Go Certificate Manager
This initiates ACME integration using the Let's Encrypt staging API.
```bash
go run certs/certifications.go
```

---

### Step 4: Verify SSL/TLS Certificates

1. Access `https://staging.myproject.com` in a browser.
2. Confirm the browser displays a certificate issued by **Fake LE Intermediate X1** (indicating the staging environment).
3. Check Traefik logs to confirm the challenge and issuance process succeeded:
   ```bash
   sudo journalctl -u traefik -f
   ```

---

## Maintenance

### Renew Certificates
Traefik handles automatic renewal, but you can force renewal manually for testing:
```bash
sudo traefik --configFile=/path/to/traefik.yaml
```

### Debugging
- **DNS Resolution Issues**: Ensure DNS records point to the correct server IP.
- **Port Issues**: Confirm ports 80 and 443 are open on your firewall.
- **ACME Errors**: Check Traefik logs for error messages and adjust configurations.

---

## Additional Notes
- Switch to the Let's Encrypt **production environment** when confident in your setup. Update `caServer` in `traefik.yaml`:
  ```yaml
  caServer: "https://acme-v02.api.letsencrypt.org/directory"
  ```
- Regularly monitor certificates and renewals to ensure consistent security.

---

## Acknowledgements
- [Traefik Documentation](https://doc.traefik.io/traefik/)
- [Lego ACME Library](https://github.com/go-acme/lego)
- [Let's Encrypt Documentation](https://letsencrypt.org/docs/)

---

## License
This project is licensed under the MIT License. See the `LICENSE` file for details.

