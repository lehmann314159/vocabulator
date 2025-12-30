# Production Deployment Guide

This guide walks through deploying Vocabulator (and future apps) on AWS Lightsail with Caddy as a reverse proxy.

## Architecture

```
                    ┌─────────────────────────────────────────┐
                    │           AWS Lightsail                 │
                    │                                         │
Internet ──────────►│  ┌─────────┐                            │
                    │  │  Caddy  │ :80/:443                   │
                    │  └────┬────┘                            │
                    │       │                                 │
                    │       ├──► vocabulator:8080             │
                    │       ├──► app2:8081                    │
                    │       └──► app3:8082                    │
                    │                                         │
                    └─────────────────────────────────────────┘
```

## Prerequisites

- AWS account
- Domain name with DNS access
- Docker Hub account (or use ECR)

## Step 1: Create Lightsail Instance

1. Go to [AWS Lightsail](https://lightsail.aws.amazon.com)
2. Click **Create instance**
3. Select:
   - **Platform:** Linux/Unix
   - **Blueprint:** OS Only → **Ubuntu 24.04 LTS**
   - **Plan:** $3.50/month (512 MB RAM) or $5/month (1 GB RAM) recommended
   - **Name:** `apps-server` (or your preference)
4. Click **Create instance**

## Step 2: Configure Networking

1. Go to your instance → **Networking** tab
2. Under **IPv4 Firewall**, add rules:
   - HTTP (80)
   - HTTPS (443)
3. Create a **Static IP** and attach it to your instance

## Step 3: Configure DNS

Add these DNS records at your domain registrar:

| Type | Name | Value |
|------|------|-------|
| A | vocabulator | YOUR_STATIC_IP |
| A | app2 | YOUR_STATIC_IP |
| A | *.yourdomain.com | YOUR_STATIC_IP |

*Using a wildcard (*) record makes adding future apps easier.*

## Step 4: Set Up the Server

SSH into your Lightsail instance:

```bash
ssh -i ~/.ssh/your-key.pem ubuntu@YOUR_STATIC_IP
```

Install Docker and Docker Compose:

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker ubuntu

# Install Docker Compose
sudo apt install docker-compose-plugin -y

# Log out and back in for group changes
exit
```

## Step 5: Deploy the Application

SSH back in and set up the project:

```bash
ssh -i ~/.ssh/your-key.pem ubuntu@YOUR_STATIC_IP

# Create app directory
mkdir -p ~/apps
cd ~/apps

# Clone your repo (or copy files)
git clone https://github.com/lehmann314159/vocabulator.git
cd vocabulator/deploy

# Create environment file
cp .env.example .env
nano .env  # Edit with your domain and email
```

Edit `.env` with your actual values:

```bash
VOCABULATOR_DOMAIN=vocabulator.yourdomain.com
ACME_EMAIL=your-email@example.com
```

Start the services:

```bash
docker compose up -d
```

## Step 6: Verify Deployment

```bash
# Check containers are running
docker compose ps

# Check logs
docker compose logs -f

# Test the API
curl https://vocabulator.yourdomain.com/health
```

## Adding New Apps

1. Add the service to `docker-compose.yml`:

```yaml
  myapp:
    build:
      context: /path/to/myapp
    container_name: myapp
    expose:
      - "8081"
    networks:
      - web
    restart: unless-stopped
```

2. Add the domain to `Caddyfile`:

```
{$MYAPP_DOMAIN:myapp.localhost} {
    reverse_proxy myapp:8081
    encode gzip
}
```

3. Add the domain to `.env`:

```bash
MYAPP_DOMAIN=myapp.yourdomain.com
```

4. Add DNS record pointing to your static IP

5. Redeploy:

```bash
docker compose up -d --build
```

## Common Commands

```bash
# View logs
docker compose logs -f vocabulator

# Restart a service
docker compose restart vocabulator

# Rebuild and restart
docker compose up -d --build vocabulator

# Stop everything
docker compose down

# Stop and remove volumes (DELETES DATA)
docker compose down -v

# Check resource usage
docker stats
```

## Backup Database

```bash
# Create backup
docker compose exec vocabulator cp /data/vocabulator.db /data/backup.db
docker cp vocabulator:/data/backup.db ./vocabulator-backup-$(date +%Y%m%d).db

# Restore backup
docker cp ./vocabulator-backup.db vocabulator:/data/vocabulator.db
docker compose restart vocabulator
```

## Troubleshooting

### SSL Certificate Issues

Caddy automatically handles SSL, but if there are issues:

```bash
# Check Caddy logs
docker compose logs caddy

# Verify domain resolves correctly
dig vocabulator.yourdomain.com
```

### Container Won't Start

```bash
# Check logs
docker compose logs vocabulator

# Check if port is in use
sudo netstat -tlnp | grep 8080
```

### Out of Memory

If the $3.50 instance runs out of memory:

```bash
# Add swap space
sudo fallocate -l 1G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

## Estimated Costs

| Resource | Monthly Cost |
|----------|-------------|
| Lightsail ($3.50 plan) | $3.50 |
| Static IP (included) | $0.00 |
| Domain (.com) | ~$1.00 |
| **Total** | **~$4.50/month** |
