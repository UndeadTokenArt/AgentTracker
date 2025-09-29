# DnD Initiative Tracker (Gin + WebSockets)

A simple modern dark-mode initiative tracker built with Go (Gin), Bootstrap, and WebSockets. Players join with a group code, the first becomes DM, and everyone sees live updates.

## Features
- Create/join groups with a short code
- Players add initiative or bonus and roll server-side
- DM adds monsters, tracks HP privately, and can reorder via drag-and-drop
- Next Turn and round tracking with active highlight
- Live sync using WebSockets
- Bootstrap Darkly theme

## Run

### Local Development
1. Install Go 1.21+
2. From repo root:

   ```bash
   go mod tidy
   go run .
   ```
3. Open http://localhost:8080

### Docker Deployment
```bash
# Build and run with Docker
docker build -t agenttracker .
docker run -p 80:80 agenttracker

# Or use docker-compose
docker-compose up -d
```

### Cloud Deployment Security Rules
Configure these inbound rules for your cloud instance:

**AWS Security Group / GCP Firewall / Azure NSG:**
- **Type:** HTTP
- **Protocol:** TCP  
- **Port:** 80
- **Source:** 0.0.0.0/0 (IPv4)
- **Description:** Allow HTTP traffic from anywhere

**For production, consider:**
- Use HTTPS (port 443) with SSL certificates
- Place behind a reverse proxy (NGINX/CloudFlare)
- Restrict database access to internal networks only

## Notes
- In-memory store only; data resets on restart.
- Players cannot set initiative below 0; players are ordered before monsters on ties.
- Monster HP is hidden from players; visible to DM only.
- WebSocket connections use the same port as HTTP.