# Z44-Tunnel

Z44-Tunnel is a lightweight, reverse TCP tunnel built with Go for securely exposing services from a private network (home lab, behind NAT/CGNAT) through a public VPS without opening inbound ports on the client side.

It uses mutual TLS (mTLS) for strong identity-based authentication and yamux for efficient stream multiplexing over a single outbound TCP connection

---

## ✨ Features

- **Pure mTLS security** (private CA, client & server authentication)
- **Reverse tunnel** (client initiates outbound connection only)
- **yamux multiplexing** (multiple streams over one TCP connection)
- **Port mapping via JSON config**
- **Reconnect & keepalive logic** for long-lived stability
- **No inbound ports required on the client** (NAT/CGNAT friendly)
- Designed for **self-hosting, homelabs, and private services**

---

## 🧠 Architecture Overview

```bash
+-------------------+       TLS (mTLS)        +-------------------+
|                   |  -------------------->  |                   |
|       Client      |                         |       Server      |
|    (Home / LAN)   |  <--------------------  |        (VPS)      |
|                   |     Single TCP Conn     |                   |
+-------------------+                         +-------------------+
          ▲                                             ▲
          |                                             |
          |                                             |
   Local services                                Reverse proxy
  (App1 / WS, etc)                              (nginx / caddy)
```

- The **client** connects outbound to the VPS
- A **single TCP+TLS connection** is established
- **yamux** multiplexes multiple logical streams
- The **server listens on localhost ports** and forwards traffic through the tunnel

---

## 🔐 Security Model

Z44-Tunnel uses **true mutual TLS (mTLS)**:

- A private Certificate Authority (CA) signs both client and server certificates
- The **server requires and verifies** the client certificate
- The **client verifies** the server certificate (SAN-based verification)
- TLS 1.3 is used by default (Go standard library)

This provides:

- Strong mutual authentication
- MITM protection
- Encrypted transport for all tunneled traffic

---

## 📂 Project Structure

```bash
.
├── client/
│   ├── client.go       # Tunnel client (runs in private network)
│   └── config.json     # Port mappings & server address
│
├── server/
│   └── server.go       # Tunnel server (runs on VPS)
│
├── utils/
│   └── gen_certs.go    # Private CA + cert generation utility
.
```

---

## ⚙️ Configuration

### client/config.json

```json
{
  "server_addr": "YOUR_VPS_IP_OR_DOMAIN",
  "tunnel_port": 49153,
  "mappings": [
    {
      "listen_port": 8920,
      "target_host": "192.168.1.10",
      "target_port": 8920
    }
  ]
}
```

- `server_addr` must match the **SAN** in the server certificate
- The client initiates the tunnel to `server_addr:tunnel_port`
- `listen_port` is bound on the VPS **localhost only**

---

## 🔑 Certificate Generation

Generate certificates using the provided utility:

```bash
SERVER_ADDR=YOUR_VPS_IP_OR_DOMAIN go run utils/gen_certs.go
```

This creates:

- `certs/ca.pem`
- `certs/server-cert.pem`
- `certs/server-key.pem`
- `certs/client-cert.pem`
- `certs/client-key.pem`

---

## 🏗️ Building

Build static binaries for client and server:

```bash
go build -o z44-client ./client
go build -o z44-server ./server
```

(Optional) Cross-compile for Linux (VPS-friendly):

```bash
GOOS=linux GOARCH=amd64 go build -o z44-client ./client
GOOS=linux GOARCH=amd64 go build -o z44-server ./server
```

---

## 🚀 Running

### On the VPS (server)

```bash
go run server/server.go
```

### On the private machine (client)

```bash
go run client/client.go
```

Once connected, services mapped in `config.json` become available on the VPS via `127.0.0.1:<listen_port>`.

---

## 🧩 Typical Use Cases

- Homelab behind **CGNAT**
- Secure access to **dashboards, admin panels, dev services**
- Expose **Jellyfin / Plex** from home without port forwarding
- Lightweight alternative to VPNs for service-level exposure

---

## ⚠️ Notes on yamux

`yamux` is used strictly as a **stream multiplexer**. Default configurations are explicitly overridden with keepalive and timeouts to avoid stalled connections on dead peers.

---

## Credits

- **[Zinadin Zidan](https://github.com/ZIDAN44)** --- Developer & creator
