# Deployment Guide

This guide covers deploying Lukaut to a VPS using Docker and GitHub Actions.

## Architecture Overview

```
                                    +------------------+
                                    |   Cloudflare     |
                                    |   (DNS + CDN)    |
                                    +--------+---------+
                                             |
                                             v
+----------------------------------------------------------------------------+
|                              VPS (Ubuntu 22.04+)                           |
|  +----------------------------------------------------------------------+  |
|  |                         Docker Network                               |  |
|  |                                                                      |  |
|  |   +-----------+      +-----------+      +----------------------+     |  |
|  |   |   Caddy   |----->|    App    |----->|     PostgreSQL       |     |  |
|  |   | (Reverse  |      | (Go + htmx|      |    (Database)        |     |  |
|  |   |  Proxy)   |      |  Worker)  |      |                      |     |  |
|  |   +-----------+      +-----------+      +----------------------+     |  |
|  |        |                   |                                         |  |
|  |        | :80/:443          |                                         |  |
|  |        v                   v                                         |  |
|  +----------------------------------------------------------------------+  |
|                               |                                            |
|                               v                                            |
|                        External Services                                   |
|                     - Cloudflare R2 (Storage)                              |
|                     - Postmark (Email)                                     |
|                     - Anthropic (AI)                                       |
+----------------------------------------------------------------------------+
```

## Prerequisites

1. **VPS Requirements**
   - Ubuntu 22.04+ or Debian 12+
   - Minimum 1GB RAM, 1 vCPU (2GB RAM recommended)
   - 20GB storage minimum

2. **External Services**
   - Domain name with DNS pointing to VPS
   - Cloudflare R2 bucket configured
   - Postmark account for transactional email
   - Anthropic API key

3. **GitHub Repository**
   - Repository pushed to GitHub
   - Access to repository settings for secrets

## Initial VPS Setup

### Option 1: Automated Setup

```bash
# SSH into your VPS as root
ssh root@your-vps-ip

# Download and run setup script
curl -sSL https://raw.githubusercontent.com/your-username/lukaut/main/scripts/vps-setup.sh | bash
```

### Option 2: Manual Setup

```bash
# Update system
apt update && apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com | sh
systemctl enable docker

# Install Docker Compose plugin
apt install docker-compose-plugin

# Create deploy user
useradd -m -s /bin/bash deploy
usermod -aG docker deploy

# Create deployment directory
mkdir -p /opt/lukaut/backups
chown -R deploy:deploy /opt/lukaut

# Configure firewall
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh
ufw allow http
ufw allow https
ufw enable
```

### Configure SSH Access

```bash
# On your local machine, generate a deploy key (if you don't have one)
ssh-keygen -t ed25519 -C "lukaut-deploy" -f ~/.ssh/lukaut_deploy

# Copy public key to VPS
ssh-copy-id -i ~/.ssh/lukaut_deploy.pub deploy@your-vps-ip

# Test connection
ssh -i ~/.ssh/lukaut_deploy deploy@your-vps-ip
```

## GitHub Secrets Configuration

Navigate to your repository's Settings > Secrets and Variables > Actions and add:

| Secret Name | Description | Example |
|-------------|-------------|---------|
| `VPS_HOST` | VPS IP or hostname | `203.0.113.10` |
| `VPS_USER` | SSH username | `deploy` |
| `VPS_SSH_KEY` | Private SSH key (full content) | `-----BEGIN OPENSSH PRIVATE KEY-----...` |
| `VPS_DEPLOY_PATH` | Deployment directory | `/opt/lukaut` |

Also add these as repository variables (Settings > Variables):

| Variable Name | Description | Example |
|---------------|-------------|---------|
| `APP_URL` | Full application URL | `https://lukaut.example.com` |
| `APP_DOMAIN` | Domain without protocol | `lukaut.example.com` |

## Production Environment Configuration

```bash
# SSH into VPS as deploy user
ssh deploy@your-vps-ip

# Navigate to deployment directory
cd /opt/lukaut

# Create production environment file
nano .env.production
```

Copy the contents from `.env.production.example` and fill in your values:

```bash
# Required values to configure:
DOMAIN=lukaut.example.com
BASE_URL=https://lukaut.example.com
POSTGRES_PASSWORD=your-strong-database-password
SMTP_USERNAME=your-postmark-token
SMTP_PASSWORD=your-postmark-token
R2_ACCOUNT_ID=your-cloudflare-account-id
R2_ACCESS_KEY_ID=your-r2-access-key
R2_SECRET_ACCESS_KEY=your-r2-secret-key
ANTHROPIC_API_KEY=sk-ant-api-xxx
VALID_INVITE_CODES=YOUR,INVITE,CODES
```

## First Deployment

### Option 1: Via GitHub Actions (Recommended)

1. Push to the `main` branch
2. The deploy workflow will automatically:
   - Build the Docker image
   - Push to GitHub Container Registry
   - Deploy to VPS

### Option 2: Manual Deployment

```bash
# On VPS as deploy user
cd /opt/lukaut

# Copy files from repository (or scp them)
# - docker-compose.prod.yml
# - Caddyfile
# - .env.production

# Log in to GitHub Container Registry
echo $GITHUB_TOKEN | docker login ghcr.io -u YOUR_USERNAME --password-stdin

# Pull and start services
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d

# Check logs
docker compose -f docker-compose.prod.yml logs -f
```

## Monitoring & Maintenance

### View Logs

```bash
# All services
docker compose -f docker-compose.prod.yml logs -f

# Specific service
docker compose -f docker-compose.prod.yml logs -f app

# Last 100 lines
docker compose -f docker-compose.prod.yml logs --tail=100 app
```

### Health Check

```bash
# Application health
curl -f http://localhost:8080/health

# Via Caddy (should return 200)
curl -f https://your-domain.com/health
```

### Database Operations

```bash
# Connect to PostgreSQL
docker compose -f docker-compose.prod.yml exec db psql -U lukaut -d lukaut

# Manual backup
docker compose -f docker-compose.prod.yml --profile backup run --rm db-backup

# List backups
ls -la backups/

# Restore from backup
./scripts/restore-backup.sh backups/lukaut_YYYYMMDD_HHMMSS.dump
```

### Update Deployment

```bash
# Pull latest images
docker compose -f docker-compose.prod.yml pull

# Restart with new images
docker compose -f docker-compose.prod.yml up -d

# Clean up old images
docker image prune -af --filter "until=24h"
```

## Troubleshooting

### Application Won't Start

```bash
# Check container status
docker compose -f docker-compose.prod.yml ps

# Check application logs
docker compose -f docker-compose.prod.yml logs app

# Common issues:
# - DATABASE_URL incorrect: Check .env.production
# - Port already in use: Check for conflicting services
# - Missing environment variables: Verify all required vars are set
```

### Database Connection Issues

```bash
# Test database connectivity
docker compose -f docker-compose.prod.yml exec db pg_isready -U lukaut

# Check database logs
docker compose -f docker-compose.prod.yml logs db
```

### SSL Certificate Issues

Caddy automatically obtains certificates. If issues occur:

```bash
# Check Caddy logs
docker compose -f docker-compose.prod.yml logs caddy

# Ensure DNS is pointing to VPS
dig +short your-domain.com

# Verify ports 80/443 are accessible
curl -I http://your-domain.com
```

### Out of Disk Space

```bash
# Check disk usage
df -h

# Clean Docker resources
docker system prune -af --volumes

# Remove old backups
find /opt/lukaut/backups -name "*.dump" -mtime +30 -delete
```

## Security Considerations

1. **Keep secrets secure**: Never commit `.env.production` to version control
2. **Update regularly**: Enable automatic security updates
3. **Monitor access**: Review SSH logs (`/var/log/auth.log`)
4. **Backup regularly**: Automated backups run daily at 2 AM UTC
5. **Use strong passwords**: Especially for database and API keys

## Rollback Procedure

If a deployment fails:

```bash
# List available image tags
docker images ghcr.io/your-username/lukaut

# Update IMAGE_TAG in .env.production to previous version
# For example: IMAGE_TAG=abc1234

# Redeploy
docker compose -f docker-compose.prod.yml up -d
```

## Scaling Considerations

For handling increased load:

1. **Vertical scaling**: Upgrade VPS resources
2. **Worker concurrency**: Increase `WORKER_CONCURRENCY` environment variable
3. **Database**: Consider managed PostgreSQL (e.g., Supabase, Neon)
4. **CDN**: Cloudflare handles static asset caching

For significant scale, consider migrating to:
- Kubernetes (managed or self-hosted)
- Cloud Run / App Engine
- Fly.io for global distribution
