# EchPlus

[中文](README_CN.md) | English

> ⚠️ **Warning**: This project is still under development. Features may be incomplete or subject to change.

EchPlus is a proxy tool based on ECH (Encrypted Client Hello) technology, supporting SOCKS5 and HTTP proxy protocols.

## ✨ Features

- 🔐 ECH-based technology for enhanced privacy protection
- 🖥️ Cross-platform desktop client (Windows / macOS / Linux)
- 💻 Command-line client, suitable for server deployment
- 🌐 Supports SOCKS5 and HTTP proxy protocols
- 🚦 Multiple routing modes: Global Proxy / Bypass China Mainland / No Proxy Change
- ⚙️ Custom DoH server support

## 🚀 Quick Start

### Command-Line Client

```bash
# Download the binary for your platform and run
./echplus-client -l 127.0.0.1:30000 -f your-worker.workers.dev:443 -token your-token
```

**Parameters:**

| Parameter  | Environment Variable | Default Value              | Description              |
| ---------- | -------------------- | -------------------------- | ------------------------ |
| `-l`       | `ECHPLUS_LISTEN`     | `127.0.0.1:30000`          | Proxy listen address     |
| `-f`       | `ECHPLUS_SERVER`     | -                          | Server address (required)|
| `-ip`      | `ECHPLUS_SERVER_IP`  | -                          | Specify server IP        |
| `-token`   | `ECHPLUS_TOKEN`      | `147258369`                | Authentication token     |
| `-dns`     | `ECHPLUS_DNS`        | `dns.alidns.com/dns-query` | DoH server               |
| `-ech`     | `ECHPLUS_ECH_DOMAIN` | `cloudflare-ech.com`       | ECH query domain         |
| `-routing` | `ECHPLUS_ROUTING`    | `global`                   | Routing mode             |

**Routing Modes:**

- `global` - Global proxy
- `bypass_cn` - Bypass China Mainland
- `none` - No proxy change

### Desktop Client

Download the installer for your platform from [Releases](https://github.com/atticus6/echPlus/releases).

## 🛠️ Development

### Requirements

- Go 1.25+
- Node.js 18+
- Bun 1.3+
- Wails3 (for desktop client)

### Build Command-Line Client

```bash
cd apps/client
go build -o echplus-client .
```

### Build Desktop Client

```bash
cd apps/desktop
wails3 build
```

### Development Mode

```bash
cd apps/desktop
wails3 dev
```

## 📚 Documentation

For detailed documentation, visit: https://echplus.netlify.app/

## 📄 License

This project is open-sourced under the [MIT License](LICENSE).

## 🤝 Contributing

Issues and Pull Requests are welcome!
