#!/usr/bin/env bash
# Setup script for Forgejo Actions Runner on Staging server
# Run as root: sudo bash scripts/setup-forgejo-runner.sh
set -euo pipefail

# --- Configuration ---
RUNNER_VERSION="6.3.1"
RUNNER_USER="forgejo-runner"
RUNNER_HOME="/var/lib/forgejo-runner"
ENV_FILE="${XELANOTE_ENV_FILE:-/home/$(whoami)/.xelanote.env}"
DOCKER_NETWORK="${DOCKER_NETWORK:-bridge}"

# --- Helper ---
info()  { echo -e "\033[1;34m[INFO]\033[0m  $*"; }
warn()  { echo -e "\033[1;33m[WARN]\033[0m  $*"; }
error() { echo -e "\033[1;31m[ERROR]\033[0m $*"; exit 1; }

# --- Root check ---
if [ "$(id -u)" -ne 0 ]; then
  error "This script must be run as root (use sudo)"
fi

# --- Pre-flight checks ---
info "Running pre-flight checks..."

for cmd in git curl docker; do
  if ! command -v "$cmd" > /dev/null 2>&1; then
    error "Required command not found: $cmd"
  fi
done

if ! docker info > /dev/null 2>&1; then
  error "Docker daemon is not running"
fi

if ! docker network inspect "$DOCKER_NETWORK" > /dev/null 2>&1; then
  warn "Docker network '$DOCKER_NETWORK' does not exist yet (will be needed at deploy time)"
fi

if [ ! -f "$ENV_FILE" ]; then
  error "Environment file not found: $ENV_FILE"
fi

# --- Detect architecture ---
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH_SUFFIX="amd64" ;;
  aarch64) ARCH_SUFFIX="arm64" ;;
  *)       error "Unsupported architecture: $ARCH" ;;
esac

# --- Download and install runner ---
info "Installing forgejo-runner v${RUNNER_VERSION} (${ARCH_SUFFIX})..."
DOWNLOAD_URL="https://code.forgejo.org/forgejo/runner/releases/download/v${RUNNER_VERSION}/forgejo-runner-${RUNNER_VERSION}-linux-${ARCH_SUFFIX}"

if [ -f /usr/local/bin/forgejo-runner ]; then
  CURRENT_VERSION=$(/usr/local/bin/forgejo-runner --version 2>/dev/null | grep -oP '[\d.]+' | head -1 || echo "unknown")
  warn "forgejo-runner already installed (version: $CURRENT_VERSION). Overwriting..."
fi

curl -fsSL "$DOWNLOAD_URL" -o /usr/local/bin/forgejo-runner
chmod 755 /usr/local/bin/forgejo-runner

INSTALLED_VERSION=$(/usr/local/bin/forgejo-runner --version 2>/dev/null || echo "unknown")
info "Installed: $INSTALLED_VERSION"

# --- Create system user ---
if id "$RUNNER_USER" > /dev/null 2>&1; then
  info "User '$RUNNER_USER' already exists"
else
  info "Creating system user '$RUNNER_USER'..."
  useradd --system --shell /usr/sbin/nologin --home-dir "$RUNNER_HOME" --create-home "$RUNNER_USER"
fi

# --- Add to docker group ---
if groups "$RUNNER_USER" | grep -q docker; then
  info "User '$RUNNER_USER' is already in the docker group"
else
  info "Adding '$RUNNER_USER' to docker group..."
  usermod -aG docker "$RUNNER_USER"
fi

# --- Set ACL on env-file ---
info "Setting POSIX ACL on $ENV_FILE..."
if ! command -v setfacl > /dev/null 2>&1; then
  warn "setfacl not found, installing acl package..."
  if command -v apt-get > /dev/null 2>&1; then
    apt-get install -y acl
  elif command -v dnf > /dev/null 2>&1; then
    dnf install -y acl
  elif command -v pacman > /dev/null 2>&1; then
    pacman -S --noconfirm acl
  else
    error "Cannot install acl package - please install manually"
  fi
fi

setfacl -m "u:${RUNNER_USER}:r" "$ENV_FILE"

# Verify env-file is readable by runner user
info "Verifying env-file access..."
if su -s /bin/bash "$RUNNER_USER" -c "cat '$ENV_FILE' > /dev/null 2>&1"; then
  info "Env-file is readable by $RUNNER_USER"
else
  error "Env-file is NOT readable by $RUNNER_USER - check parent directory permissions"
fi

# --- Create working directory ---
info "Setting up working directory: $RUNNER_HOME"
mkdir -p "$RUNNER_HOME"
chown -R "$RUNNER_USER:$RUNNER_USER" "$RUNNER_HOME"

# --- Create systemd service ---
info "Creating systemd service..."
cat > /etc/systemd/system/forgejo-runner.service << 'UNIT'
[Unit]
Description=Forgejo Actions Runner
Documentation=https://forgejo.org/docs/latest/admin/actions/
After=docker.service
Requires=docker.service

[Service]
Type=simple
User=forgejo-runner
Group=forgejo-runner
WorkingDirectory=/var/lib/forgejo-runner
ExecStart=/usr/local/bin/forgejo-runner daemon
Restart=on-failure
RestartSec=10

# Security hardening
ProtectSystem=strict
ProtectHome=read-only
ReadWritePaths=/var/lib/forgejo-runner
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
UNIT

systemctl daemon-reload
systemctl enable forgejo-runner

info "Systemd service created and enabled"

# --- Done ---
echo ""
echo "==========================================="
info "Setup complete!"
echo "==========================================="
echo ""
echo "Next steps:"
echo ""
echo "  1. Get a runner token from Forgejo:"
echo "     Repo -> Settings -> Actions -> Runners -> Create new runner"
echo ""
echo "  2. Register the runner:"
echo "     sudo -u $RUNNER_USER forgejo-runner register \\"
echo "       --instance https://<FORGEJO_URL> \\"
echo "       --token <TOKEN> \\"
echo "       --name staging-runner \\"
echo "       --labels staging:host \\"
echo "       --no-interactive"
echo ""
echo "  3. Start the runner:"
echo "     sudo systemctl start forgejo-runner"
echo ""
echo "  4. Verify:"
echo "     sudo systemctl status forgejo-runner"
echo "     sudo journalctl -u forgejo-runner -f"
echo ""
echo "  5. Enable Actions in Forgejo:"
echo "     - Repository Settings -> Features -> 'Actions' checkbox"
echo "     - app.ini: [actions] ENABLED=true"
echo "     - app.ini: DEFAULT_ACTIONS_URL=github"
echo ""
