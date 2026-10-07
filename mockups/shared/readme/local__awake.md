# Awake

Self-hosted Wake-on-LAN over the internet. Keep power-hungry machines off, and wake them from anywhere with one tap.

```
 Browser / PWA ──HTTPS──▶  Awake server (VPS)  ◀──WebSocket (outbound)──  Agent on your LAN ──magic packet──▶ PC, NAS…
                           devices, users,                                 MCU / Raspberry Pi
                           sharing, logs                                   wake + ping on request
```

- **Server** (always on, e.g. a small VPS): stores devices, MCUs and users, serves the web UI, tells agents what to do.
- **Agent** (always on, inside your LAN): a microcontroller or a Raspberry Pi that keeps an outbound connection to the server, sends magic packets and checks whether devices are up. Nothing to open on your router.
- **Userspaces**: one per person. Share the server with friends; each manages their own MCUs and devices and can share a device with someone else's userspace by slug.

## Repository

| Path | Content |
|---|---|
| [`backend/`](backend/) | Server: HTTP API, MCU gateway, SQLite. [API reference](backend/API.md) |
| [`shared/`](shared/) | Types, error codes and validation shared by backend and frontend |
| [`frontend/`](frontend/) | Web UI: Svelte 5 PWA, Material 3, English and Italian |
| [`mcu-templates/`](mcu-templates/) | Agents for specific boards + the [protocol spec](mcu-templates/PROTOCOL.md) to write your own |

## Run the server

```bash
cp .env.example .env    # optional
make up                 # docker compose up -d --build
make password           # shows the generated admin password
```

The server speaks plain HTTP on port 8080. Put a reverse proxy with HTTPS in front of it (Caddy, Traefik, nginx…) if it is reachable from the internet: userspace codes, the admin password and MCU tokens travel in every request.

WebSocket must be proxied for `/agent`. With Caddy it is automatic:

```
awake.example.com {
    reverse_proxy localhost:8080
}
```

Then set `TRUST_PROXY=true`.

### Configuration

| Variable | Default | |
|---|---|---|
| `PORT` | `8080` | Host port (compose) / listen port (bare metal) |
| `ADMIN_PASSWORD` | generated | If unset, generated on first start, stored in the data volume and printed at every start. If set, it always wins. |
| `DEFAULT_LOCALE` | `en` | UI language for users who did not pick one: `en`, `it` |
| `TRUST_PROXY` | `false` | Use `X-Forwarded-For` as client IP (rate limiting) |
| `DATA_DIR` | `/data` in Docker | SQLite database location |

## First steps

1. Open the admin panel with the admin password and create a userspace (a slug, e.g. `jacopo`). Copy its code: it is shown once.
2. Log in with the code. Create an MCU and copy its token.
3. Run an agent in your LAN with that token (e.g. [the Node.js agent](mcu-templates/nodejs/) on a Raspberry Pi).
4. Add a device: name, MAC, optionally IP (to see whether it is on) and a TCP port (for machines that ignore ping), and assign it to the MCU.
5. Wake it.
