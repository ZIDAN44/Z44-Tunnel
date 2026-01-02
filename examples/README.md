# Z44 Tunnel Systemd Service Setup

This directory contains systemd service files for running Z44 Tunnel as a system service on Linux.

## Prerequisites

- Linux system with systemd
- Z44 Tunnel binaries built and installed to `/opt/z44/` (See the [Building](../README.md#️-building) instructions)
- Certificates generated and placed in `/opt/z44/certs/`
- Client configuration file at `/opt/z44/config.json` (for client setup - must be in the working directory)

## Certificate Generation

Before setting up the services, you need to generate certificates. (See the [Certificate Generation](../README.md#-certificate-generation) instructions)

**Copy certificates to installation directory:**

```bash
sudo mkdir -p /opt/z44/certs
sudo cp certs/*.pem /opt/z44/certs/
```

## Installation

### Client Setup

1. **Create directory and copy the client binary:**

   ```bash
   sudo mkdir -p /opt/z44
   sudo cp z44-client /opt/z44/client
   ```

   **Copy required certificates:**

   ```bash
   sudo mkdir -p /opt/z44/certs
   sudo cp certs/ca.pem certs/client-cert.pem certs/client-key.pem /opt/z44/certs/
   ```

   **Copy client configuration:**

   ```bash
   sudo cp client/config.json /opt/z44/config.json
   sudo chmod 600 /opt/z44/config.json
   ```

2. **Copy the service file:**

   ```bash
   sudo cp z44-client.service /etc/systemd/system/
   ```

3. **Create system user and set permissions:**

   ```bash
   sudo useradd --system --no-create-home --shell /usr/sbin/nologin z44 || true
   sudo chown -R z44:z44 /opt/z44
   sudo chmod +x /opt/z44/client
   sudo chmod 700 /opt/z44/certs
   sudo chmod 600 /opt/z44/certs/ca.pem /opt/z44/certs/client-cert.pem /opt/z44/certs/client-key.pem
   sudo chmod 600 /opt/z44/config.json
   ```

4. **Enable and start the service:**

   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable --now z44-client
   ```

5. **Check service status:**

   ```bash
   sudo systemctl status z44-client --no-pager
   ```

6. **View logs:**

   ```bash
   journalctl -u z44-client -f
   ```

### Server Setup

1. **Create directory and copy the server binary:**

   ```bash
   sudo mkdir -p /opt/z44
   sudo cp z44-server /opt/z44/server
   ```

   **Copy required certificates:**

   ```bash
   sudo mkdir -p /opt/z44/certs
   sudo cp certs/ca.pem certs/server-cert.pem certs/server-key.pem /opt/z44/certs/
   ```

2. **Copy the service file:**

   ```bash
   sudo cp z44-server.service /etc/systemd/system/
   ```

3. **Create system user and set permissions:**

   ```bash
   sudo useradd --system --no-create-home --shell /usr/sbin/nologin z44 || true
   sudo chown -R z44:z44 /opt/z44
   sudo chmod +x /opt/z44/server
   sudo chmod 700 /opt/z44/certs
   sudo chmod 600 /opt/z44/certs/ca.pem /opt/z44/certs/server-cert.pem /opt/z44/certs/server-key.pem
   ```

4. **Enable and start the service:**

   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable --now z44-server
   ```

5. **View logs:**

   ```bash
   journalctl -u z44-server -f
   ```

## Service Management

### Common Commands

```bash
# Start the service
sudo systemctl start z44-client    # or z44-server

# Stop the service
sudo systemctl stop z44-client     # or z44-server

# Restart the service
sudo systemctl restart z44-client  # or z44-server

# Check service status
sudo systemctl status z44-client   # or z44-server

# View logs
journalctl -u z44-client -f        # or z44-server
```

## Service Configuration

Both service files are configured with:

- **User/Group:** Runs as `z44:z44` system user (non-login)
- **Working Directory:** `/opt/z44` (required for certificate paths)
- **Restart Policy:** Always restart on failure with 2-second delay
- **Start Delay:** 3-second delay to avoid network race conditions during boot
- **Security:** `NoNewPrivileges` and `PrivateTmp` enabled
- **Logging:** Outputs to systemd journal

## Notes

- The service files use `StartLimitIntervalSec=0` to prevent systemd from rate-limiting restarts
- Services automatically restart on failure with a 2-second delay
- All output is logged to the systemd journal (view with `journalctl`)
- The `z44` user is created as a system user without a home directory and cannot log in
