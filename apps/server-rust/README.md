# EchPlus Server (Rust)

VLESS WebSocket 代理服务器的 Rust 实现。

## 功能特性

- VLESS 协议支持
- WebSocket 传输
- TCP 代理
- UUID 认证
- 健康检查端点

## 构建

```bash
cargo build --release
```

## 运行

```bash
# 使用默认配置
cargo run --release

# 自定义配置
cargo run --release -- --uuid "your-uuid-here" --port 8080

# 使用环境变量
UUID="your-uuid-here" PORT=8080 cargo run --release
```

## Docker

```bash
# 构建镜像
docker build -t echplus-server .

# 运行容器
docker run -p 3325:3325 -e UUID="your-uuid-here" echplus-server
```

## 配置

- `--uuid` / `UUID`: VLESS UUID (默认: 147258369-1234-5678-9abc-def012345678)
- `--port` / `PORT`: 服务器端口 (默认: 3325)

## API

- `GET /`: WebSocket 升级端点
- `GET /health`: 健康检查
