#!/bin/bash
# =============================================================================
# VPS Initial Setup Script for Lukaut
# =============================================================================
# Run this script on a fresh Fedora 39+ VPS to prepare it for deployment.
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/your-username/lukaut/main/scripts/vps-setup.sh | sudo bash
#
# Or copy to VPS and run:
#   chmod +x vps-setup.sh
#   sudo ./vps-setup.sh
# =============================================================================

set -euo pipefail

# Configuration
DEPLOY_USER="deploy"
DEPLOY_PATH="/opt/lukaut"
BACKUP_PATH="/opt/lukaut/backups"

echo "=============================================="
echo "Lukaut VPS Setup Script"
echo "=============================================="

# Check if running as root
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root (sudo)"
  exit 1
fi

# Update system
echo "[1/8] Updating system packages..."
dnf upgrade -y --refresh

# Install required packages
echo "[2/8] Installing required packages..."
dnf install -y \
  ca-certificates \
  curl \
  gnupg2 \
  firewalld \
  fail2ban \
  dnf-automatic

# Install Docker
echo "[3/8] Installing Docker..."
if ! command -v docker &> /dev/null; then
  dnf -y install dnf-plugins-core
  dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo
  dnf install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  systemctl enable docker
  systemctl start docker
else
  echo "Docker already installed"
fi

# Install Docker Compose plugin (included above, but ensure it's present)
echo "[4/8] Verifying Docker Compose..."
docker compose version

# Create deploy user
echo "[5/8] Creating deploy user..."
if ! id "$DEPLOY_USER" &>/dev/null; then
  useradd -m -s /bin/bash "$DEPLOY_USER"
  usermod -aG docker "$DEPLOY_USER"
  echo "Deploy user created: $DEPLOY_USER"
else
  echo "Deploy user already exists"
  usermod -aG docker "$DEPLOY_USER"
fi

# Create deployment directory
echo "[6/8] Creating deployment directory..."
mkdir -p "$DEPLOY_PATH"
mkdir -p "$BACKUP_PATH"
chown -R "$DEPLOY_USER:$DEPLOY_USER" "$DEPLOY_PATH"

# Configure firewall
echo "[7/8] Configuring firewall..."
systemctl enable firewalld
systemctl start firewalld
firewall-cmd --permanent --add-service=ssh
firewall-cmd --permanent --add-service=http
firewall-cmd --permanent --add-service=https
firewall-cmd --reload

# Configure fail2ban
echo "[8/8] Configuring fail2ban..."
cat > /etc/fail2ban/jail.local << 'EOF'
[DEFAULT]
bantime = 1h
findtime = 10m
maxretry = 5

[sshd]
enabled = true
port = ssh
filter = sshd
backend = systemd
maxretry = 3
EOF

systemctl enable fail2ban
systemctl restart fail2ban

# Enable automatic security updates
cat > /etc/dnf/automatic.conf << 'EOF'
[commands]
upgrade_type = security
random_sleep = 0
download_updates = yes
apply_updates = yes

[emitters]
emit_via = stdio

[command]
command_format = cat

[base]
debuglevel = 1
EOF

systemctl enable --now dnf-automatic.timer

echo ""
echo "=============================================="
echo "Setup Complete!"
echo "=============================================="
echo ""
echo "Next steps:"
echo ""
echo "1. Set up SSH key for deploy user:"
echo "   mkdir -p /home/$DEPLOY_USER/.ssh"
echo "   # Add your SSH public key to /home/$DEPLOY_USER/.ssh/authorized_keys"
echo "   chown -R $DEPLOY_USER:$DEPLOY_USER /home/$DEPLOY_USER/.ssh"
echo "   chmod 700 /home/$DEPLOY_USER/.ssh"
echo "   chmod 600 /home/$DEPLOY_USER/.ssh/authorized_keys"
echo ""
echo "2. Copy production environment file:"
echo "   scp .env.production $DEPLOY_USER@<your-vps>:$DEPLOY_PATH/.env.production"
echo ""
echo "3. Configure GitHub secrets:"
echo "   - DOCKERHUB_USERNAME: Your DockerHub username"
echo "   - DOCKERHUB_TOKEN: Your DockerHub access token"
echo "   - VPS_HOST: Your VPS IP address"
echo "   - VPS_USER: $DEPLOY_USER"
echo "   - VPS_SSH_KEY: Your private SSH key"
echo ""
echo "4. Push to main branch to trigger deployment"
echo ""
echo "Deployment path: $DEPLOY_PATH"
echo "Backup path: $BACKUP_PATH"
echo ""
