# 版本管理

## 发布新版本

```bash
# 1. 更新版本号
# 编辑 apps/server-rust/Cargo.toml
version = "0.1.0"

# 2. 提交更改
git add apps/server-rust/Cargo.toml
git commit -m "chore: bump server-rust version to 0.1.0"
git push origin main

# 3. 创建并推送 tag
git tag server-rust-v0.1.0
git push origin server-rust-v0.1.0
```

## 自动构建产物

推送 tag 后会自动构建：

### 二进制文件
- `echplus-server-linux-amd64`
- `echplus-server-linux-arm64`
- `echplus-server-darwin-amd64`
- `echplus-server-darwin-arm64`
- `echplus-server-windows-amd64.exe`

### Docker 镜像
- `ghcr.io/your-repo/server-rust:0.1.0`
- `ghcr.io/your-repo/server-rust:latest`

## 使用

### 下载二进制
从 GitHub Releases 页面下载对应平台的二进制文件

### 使用 Docker
```bash
docker pull ghcr.io/your-repo/server-rust:latest
docker run -p 3325:3325 -e UUID="your-uuid" ghcr.io/your-repo/server-rust:latest
```
