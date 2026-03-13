# share-info

读取当前系统环境变量并加密保存，支持：
- CLI 加密/解密
- 默认输出 Base64（便于复制粘贴）
- 纯前端页面本地解密（GitHub Pages）

## 一键运行（curl | sh）

直接下载 **latest release** 中与你系统架构匹配的二进制并执行：

```bash
curl -fsSL https://raw.githubusercontent.com/yanickxia/share-info/main/scripts/run-latest.sh | \
ENV_SNAPSHOT_PASSWORD='your-pass' sh
```

默认行为等价于：

```bash
share-info -mode encrypt -out env.snapshot.enc.b64
```

输出文件为当前目录下 `env.snapshot.enc.b64`。

## 一键运行（透传参数）

可把参数直接传给二进制：

```bash
# 加密（默认 base64）
curl -fsSL https://raw.githubusercontent.com/yanickxia/share-info/main/scripts/run-latest.sh | \
ENV_SNAPSHOT_PASSWORD='your-pass' sh -s -- -mode encrypt -out ./env.snapshot.enc.b64

# 解密
curl -fsSL https://raw.githubusercontent.com/yanickxia/share-info/main/scripts/run-latest.sh | \
ENV_SNAPSHOT_PASSWORD='your-pass' sh -s -- -mode decrypt -in ./env.snapshot.enc.b64 -out ./env.snapshot.json
```

## Release 构建产物

GitHub Action 在 tag 发布时会构建：
- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`
- `windows/amd64`
- `windows/arm64`

命名格式：
- Linux/macOS: `share-info-<os>-<arch>.tar.gz`
- Windows: `share-info-<os>-<arch>.exe.zip`

## 前端解密页面（GitHub Pages）

默认地址：

```text
https://yanickxia.github.io/share-info/
```

页面在浏览器本地解密，不会把密文上传到服务端。

## 临时上传（公共服务）

如果你需要把 **已加密** 的 `.b64` 文件临时分享，可直接：

```bash
# 0x0.st
curl -fsS -F "file=@env.snapshot.enc.b64" https://0x0.st

# transfer.sh
curl -fsS --upload-file ./env.snapshot.enc.b64 https://transfer.sh/env.snapshot.enc.b64
```

也可以一条命令“生成并上传”：

```bash
curl -fsSL https://raw.githubusercontent.com/yanickxia/share-info/main/scripts/run-latest.sh | \
ENV_SNAPSHOT_PASSWORD='your-pass' sh -s -- -mode encrypt -out ./env.snapshot.enc.b64 && \
curl -fsS -F "file=@env.snapshot.enc.b64" https://0x0.st
```

注意：以上为公共服务，链接拿到即可访问，请仅上传已加密文件并控制有效期。
