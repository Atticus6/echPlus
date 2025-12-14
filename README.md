# EchPlus

> ⚠️ **警告**: 本项目仍在开发中，功能可能不完整或存在变动。

EchPlus 是一个基于 ECH (Encrypted Client Hello) 技术的代理工具，支持 SOCKS5 和 HTTP 代理协议。

## ✨ 特性

- 🔐 基于 ECH 技术，增强隐私保护
- 🖥️ 跨平台桌面客户端 (Windows / macOS / Linux)
- 💻 命令行客户端，适合服务器部署
- 🌐 支持 SOCKS5 和 HTTP 代理协议
- 🚦 多种分流模式：全局代理 / 跳过中国大陆 / 不改变代理
- ⚙️ 支持自定义 DoH 服务器

## 🚀 快速开始

### 命令行客户端

```bash
# 下载对应平台的二进制文件后运行
./echplus-client -l 127.0.0.1:30000 -f your-worker.workers.dev:443 -token your-token
```

**参数说明：**

| 参数       | 环境变量             | 默认值                     | 说明              |
| ---------- | -------------------- | -------------------------- | ----------------- |
| `-l`       | `ECHPLUS_LISTEN`     | `127.0.0.1:30000`          | 代理监听地址      |
| `-f`       | `ECHPLUS_SERVER`     | -                          | 服务端地址 (必填) |
| `-ip`      | `ECHPLUS_SERVER_IP`  | -                          | 指定服务端 IP     |
| `-token`   | `ECHPLUS_TOKEN`      | `147258369`                | 身份验证令牌      |
| `-dns`     | `ECHPLUS_DNS`        | `dns.alidns.com/dns-query` | DoH 服务器        |
| `-ech`     | `ECHPLUS_ECH_DOMAIN` | `cloudflare-ech.com`       | ECH 查询域名      |
| `-routing` | `ECHPLUS_ROUTING`    | `global`                   | 分流模式          |

**分流模式：**

- `global` - 全局代理
- `bypass_cn` - 跳过中国大陆
- `none` - 不改变代理

### 桌面客户端

从 [Releases](https://github.com/atticus6/echPlus/releases) 下载对应平台的安装包。

## 🛠️ 开发

### 环境要求

- Go 1.25+
- Node.js 18+
- Bun 1.3+
- Wails3 (桌面客户端)

### 构建命令行客户端

```bash
cd apps/client
go build -o echplus-client .
```

### 构建桌面客户端

```bash
cd apps/desktop
wails3 build
```

### 开发模式

```bash
cd apps/desktop
wails3 dev
```

## 📄 许可证

本项目基于 [MIT License](LICENSE) 开源。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！
