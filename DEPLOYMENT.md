# Lukaut Deployment Guide

## Infrastructure Overview

| Component | Details |
|-----------|---------|
| VPS Provider | Hetzner |
| OS | Ubuntu 22.04 LTS |
| IP | 5.78.98.218 |
| Domain | lukaut.com (primary), lukaut.app (redirects to .com) |
| Reverse Proxy | Caddy (host-installed, auto-SSL) |
| Container Runtime | Docker + Docker Compose |

## Server Access

```bash
# SSH as deploy user
ssh deploy@manifoldcollective.com

# Or explicitly
ssh -i ~/.ssh/lukaut_deploy deploy@5.78.98.218
```

## Directory Structure

```
/opt/lukaut/
├── docker-compose.yml    # Production compose file
├── .env                  # Production environment variables
└── backups/              # Database backup directory
```

## CI/CD Pipeline

The deployment is fully automated via GitHub Actions:

1. **Push to main** → Triggers CI workflow
2. **CI workflow** → Runs test, lint, build jobs
3. **CI succeeds** → Triggers Deploy workflow
4. **Deploy workflow** → Builds/pushes Docker image, SSHs to VPS, pulls and restarts containers

### Manual Deployment

```bash
# Trigger deploy manually
gh workflow run deploy.yml

# Watch progress
gh run watch
```

### GitHub Secrets Required

| Secret | Description |
|--------|-------------|
| `VPS_HOST` | Server IP or hostname |
| `VPS_USER` | SSH username (`deploy`) |
| `VPS_SSH_KEY` | Private SSH key (no passphrase) |
| `DOCKERHUB_USERNAME` | DockerHub username |
| `DOCKERHUB_TOKEN` | DockerHub access token |

### GitHub Variables Required

| Variable | Value |
|----------|-------|
| `APP_URL` | `https://lukaut.com` |
| `APP_DOMAIN` | `lukaut.com` |

## Services

### Docker Containers

```bash
# View running containers
docker ps

# View logs
docker compose logs -f
docker compose logs -f app

# Restart services
docker compose down && docker compose up -d

# Pull latest and restart
docker compose pull && docker compose up -d
```

### Caddy (Reverse Proxy)

```bash
# Config location
/etc/caddy/Caddyfile

# View/edit config
sudo nano /etc/caddy/Caddyfile

# Reload after changes
sudo systemctl reload caddy

# Check status
sudo systemctl status caddy

# View logs
sudo journalctl -u caddy -f
```

**Current Caddyfile:**
```
lukaut.com {
    reverse_proxy 127.0.0.1:8080
}

lukaut.app {
    redir https://lukaut.com{uri} permanent
}
```

## Security

### Firewall (UFW)

```bash
# Check status
sudo ufw status

# Allowed ports: 22 (SSH), 80 (HTTP), 443 (HTTPS)
```

### Fail2ban

```bash
# Check status
sudo systemctl status fail2ban

# View banned IPs
sudo fail2ban-client status sshd

# Config location
/etc/fail2ban/jail.local
```

### SSH

- Root login disabled
- Password authentication disabled
- Only key-based authentication allowed
- Config: `/etc/ssh/sshd_config.d/hardening.conf`

## Database

### PostgreSQL (runs in Docker)

```bash
# Connect to database
docker exec -it lukaut-db psql -U lukaut -d lukaut

# Manual backup
docker compose --profile backup run --rm db-backup

# Backups stored in
/opt/lukaut/backups/
```

## Environment Variables

Production environment variables are stored in `/opt/lukaut/.env`. Template available at `.env.production.example` in the repo.

Key variables:
- `BASE_URL` - https://lukaut.com
- `POSTGRES_PASSWORD` - Database password
- `ANTHROPIC_API_KEY` - AI provider key
- `R2_*` - Cloudflare R2 storage credentials
- `SMTP_*` - Postmark email credentials

## Troubleshooting

### App not responding

```bash
# Check container status
docker ps

# Check app logs
docker compose logs app --tail 100

# Check if port is listening
curl http://127.0.0.1:8080/health
```

### 502 Bad Gateway

```bash
# Check if app container is running and healthy
docker ps

# Check Caddy logs
sudo journalctl -u caddy --no-pager -n 50

# Verify Caddy can reach the app
curl -v http://127.0.0.1:8080/health
```

### SSL Issues (525 error)

- Check Cloudflare SSL/TLS settings → Set to "Full" (not "Full strict")
- Verify Caddy has obtained certificates: `sudo journalctl -u caddy | grep certificate`

### SSH Connection Issues

```bash
# Test with verbose output
ssh -v -i ~/.ssh/lukaut_deploy deploy@5.78.98.218

# Check server auth log
sudo tail -f /var/log/auth.log
```

## Initial Server Setup

For setting up a new VPS from scratch:

```bash
# 1. Update system
apt update && apt upgrade -y
apt install -y curl wget git unzip htop ncdu

# 2. Create deploy user
adduser deploy
usermod -aG sudo deploy

# 3. Set up SSH keys for deploy user
su - deploy
mkdir -p ~/.ssh && chmod 700 ~/.ssh
echo "PUBLIC_KEY_HERE" >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
exit

# 4. Secure SSH
cat > /etc/ssh/sshd_config.d/hardening.conf << 'EOF'
PermitRootLogin no
PasswordAuthentication no
PubkeyAuthentication yes
AuthorizedKeysFile .ssh/authorized_keys
X11Forwarding no
MaxAuthTries 3
EOF
systemctl restart sshd

# 5. Set up firewall
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable

# 6. Install Docker
curl -fsSL https://get.docker.com | sh
usermod -aG docker deploy
systemctl enable docker

# 7. Install Fail2ban
apt install -y fail2ban
cat > /etc/fail2ban/jail.local << 'EOF'
[DEFAULT]
bantime = 1h
findtime = 10m
maxretry = 5

[sshd]
enabled = true
port = 22
filter = sshd
logpath = /var/log/auth.log
maxretry = 3
EOF
systemctl enable fail2ban
systemctl start fail2ban

# 8. Install Caddy
apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list
apt update
apt install caddy

# 9. Create app directory
mkdir -p /opt/lukaut/backups
chown -R deploy:deploy /opt/lukaut
```

Then copy files and start:
```bash
# From local machine
scp docker-compose.prod.yml deploy@server:/opt/lukaut/docker-compose.yml
scp .env.production.example deploy@server:/opt/lukaut/.env

# On server
cd /opt/lukaut
nano .env  # Edit with production values
docker compose up -d
```
