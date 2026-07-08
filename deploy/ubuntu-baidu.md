# 百度云 Ubuntu 部署步骤（前后端完整版）

> 无 Docker | 路径示例：`/opt/tools_web` | Nginx 对外 | systemd 托管后端

---

## 架构一览

```
浏览器
  ↓ :80 / :443（或独立端口 :18082，见下方端口规划）
Nginx ── 静态文件 ← tools-web-frontend/dist
  │
  └── /api/* → Go API 127.0.0.1:18080
                  └── HTTP → Python ASR 127.0.0.1:18081
                  └── ffmpeg / yt-dlp / BOS(可选)
```

---

## 端口规划（与其他服务共存）

默认避开常见的 **8080 / 8090 / 8000 / 9000**，本项目的端口分配如下：

| 组件 | 端口 | 监听地址 | 是否对外开放 | 说明 |
|------|------|----------|--------------|------|
| Nginx（推荐） | **80 / 443** | `0.0.0.0` | 是 | 与其他站点**共用 80**，用 `server_name` 区分域名 |
| Nginx（备选） | **18082** | `0.0.0.0` | 是 | 80 已被独占、无法做虚拟主机时使用 |
| Go API | **18080** | `127.0.0.1` | **否** | 仅 Nginx 反代，安全组不必放行 |
| Python ASR | **18081** | `127.0.0.1` | **否** | 仅 Go API 调用 |
| Vite 开发 | **5173** | `localhost` | 否 | 仅本机开发 |

**与其他 Web 服务共存（推荐）：** 80 端口可以跑多个站点，每个站点一个 `server { server_name xxx; }` 块，互不冲突。

**检查端口占用：**

```bash
ss -tlnp | grep -E ':(80|443|8080|8090|18080|18081|18082)\s'
```

若 **18080 / 18081** 也被占用，可在 `.env` 里改成例如 `19080` / `19081`，并同步修改 systemd 里 ASR 的 `--port` 与 Nginx 的 `proxy_pass`。

---

## 第一步：服务器初始化（一次性）

```bash
# 以 root 或 sudo 执行
apt update && apt upgrade -y
apt install -y git curl wget nginx ffmpeg \
  python3 python3-venv python3-pip build-essential vim

pip3 install yt-dlp --break-system-packages
```

### 安装 Go 1.22（不要用 apt 的 golang-go 1.18）

```bash
cd /tmp
wget https://go.dev/dl/go1.22.5.linux-amd64.tar.gz
rm -rf /usr/local/go
tar -C /usr/local -xzf go1.22.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
export PATH=$PATH:/usr/local/go/bin
go version
```

### 安装 Node.js 20（前端构建）

```bash
curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
apt install -y nodejs
node -v && npm -v
```

### 创建目录

```bash
mkdir -p /opt/tools_web/bin /var/lib/tools-web/tmp
```

---

## 第二步：拉取代码

```bash
cd /opt/tools_web

git clone https://github.com/zhouligit/tools-web-backend.git
git clone https://github.com/zhouligit/tools-web-frontend.git
```

---

## 第三步：部署后端

### 3.1 配置环境变量

```bash
cd /opt/tools_web/tools-web-backend
cp .env.example .env
vim .env
```

参考配置：

```env
ADDR=127.0.0.1:18080
FRONTEND_ORIGIN=http://你的服务器IP或域名
TEMP_DIR=/var/lib/tools-web/tmp
MAX_UPLOAD_MB=500
ASR_SERVICE_URL=http://127.0.0.1:18081
FFMPEG_PATH=ffmpeg
YTDLP_PATH=yt-dlp

# 暂不用 BOS 可设 false
BOS_ENABLED=false
BOS_ENDPOINT=https://bj.bcebos.com
BOS_BUCKET=
BOS_ACCESS_KEY=
BOS_SECRET_KEY=
```

### 3.2 编译 Go API

```bash
cd /opt/tools_web/tools-web-backend
export PATH=$PATH:/usr/local/go/bin
export GOPROXY=https://goproxy.cn,direct

go mod tidy
go build -o /opt/tools_web/bin/server ./cmd/server
```

### 3.3 安装 Python ASR 服务

```bash
cd /opt/tools_web/tools-web-backend/asr-service
rm -rf .venv
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple
deactivate
```

> 若 venv 失败：先执行 `apt install -y python3-venv python3.10-venv`

### 3.4 注册 systemd — ASR 服务

```bash
cat > /etc/systemd/system/tools-asr.service <<'EOF'
[Unit]
Description=Tools Web ASR (faster-whisper)
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/tools_web/tools-web-backend/asr-service
Environment=WHISPER_MODEL=medium
Environment=WHISPER_DEVICE=cpu
Environment=WHISPER_COMPUTE_TYPE=int8
ExecStart=/opt/tools_web/tools-web-backend/asr-service/.venv/bin/uvicorn main:app --host 127.0.0.1 --port 18081
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
```

有 GPU 时在 `[Service]` 里改为：

```ini
Environment=WHISPER_DEVICE=cuda
Environment=WHISPER_COMPUTE_TYPE=float16
Environment=WHISPER_MODEL=large-v3
```

### 3.5 注册 systemd — Go API

```bash
cat > /etc/systemd/system/tools-api.service <<'EOF'
[Unit]
Description=Tools Web Go API
After=network.target tools-asr.service
Requires=tools-asr.service

[Service]
Type=simple
WorkingDirectory=/opt/tools_web/tools-web-backend
EnvironmentFile=/opt/tools_web/tools-web-backend/.env
ExecStart=/opt/tools_web/bin/server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable tools-asr tools-api
systemctl start tools-asr tools-api
```

### 3.6 验证后端

```bash
systemctl status tools-asr tools-api
curl http://127.0.0.1:18081/health
curl http://127.0.0.1:18080/api/v1/health
journalctl -u tools-asr -n 30 --no-pager
journalctl -u tools-api -n 30 --no-pager
```

---

## 第四步：部署前端

### 4.1 构建静态资源

```bash
cd /opt/tools_web/tools-web-frontend
npm ci
npm run build
```

产物目录：`/opt/tools_web/tools-web-frontend/dist`

### 4.2 配置 Nginx

把 `你的域名或IP` 换成实际值，例如 `1.2.3.4` 或 `tools.example.com`：

```bash
cat > /etc/nginx/sites-available/tools-web <<'EOF'
server {
    listen 80;
    server_name 你的域名或IP;

    root /opt/tools_web/tools-web-frontend/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:18080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        client_max_body_size 500m;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }
}
EOF

ln -sf /etc/nginx/sites-available/tools-web /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default
nginx -t && systemctl reload nginx
```

**若 80 已被其他服务独占、无法加虚拟主机**，改用独立端口 **18082**（安全组需放行 18082）：

```bash
cat > /etc/nginx/sites-available/tools-web <<'EOF'
server {
    listen 18082;
    server_name _;

    root /opt/tools_web/tools-web-frontend/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:18080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        client_max_body_size 500m;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }
}
EOF
```

访问地址变为：`http://你的IP:18082`

### 4.3 百度云安全组

在百度云控制台放行：

- **80**（HTTP）
- **443**（HTTPS，若配置证书）

无需对外开放 18080、18081（仅本机访问）。

### 4.4 验证前端

浏览器访问：`http://你的服务器IP`  
测试「音视频转文字」上传或粘贴链接。

---

## 第五步：开启 HTTPS（可选）

```bash
apt install -y certbot python3-certbot-nginx
certbot --nginx -d tools.example.com
```

---

## 日常更新命令

### 更新后端

```bash
cd /opt/tools_web/tools-web-backend
git pull
export PATH=$PATH:/usr/local/go/bin
export GOPROXY=https://goproxy.cn,direct
go build -o /opt/tools_web/bin/server ./cmd/server

# requirements.txt 有变化时
cd asr-service && source .venv/bin/activate && pip install -r requirements.txt && deactivate && cd ..

systemctl restart tools-asr tools-api
```

### 更新前端

```bash
cd /opt/tools_web/tools-web-frontend
git pull
npm ci
npm run build
systemctl reload nginx
```

---

## 常见问题

| 现象 | 处理 |
|------|------|
| `go: command not found` | `export PATH=$PATH:/usr/local/go/bin` 或重装 Go |
| `ensurepip is not available` | `apt install -y python3-venv` |
| `p: command not found` | 应为 `cp .env.example .env` |
| 转写一直 processing | `journalctl -u tools-asr -f` 看模型是否加载完 |
| 上传失败/超时 | 检查 Nginx `client_max_body_size`、安全组 |
| 端口被占用 | `ss -tlnp` 查看；改 `.env` 的 `ADDR` / `ASR_SERVICE_URL` 及 systemd 中 ASR `--port` |

---

## 资源建议

| 配置 | CPU | 内存 | 说明 |
|------|-----|------|------|
| 最低 | 2 核 | 4 GB | CPU 转写较慢 |
| 推荐 | 4 核 | 8 GB | medium 模型 |
| 高配 | 8 核+ | 16 GB+ | large-v3 / GPU |

---

## 检查清单

- [ ] Go、Node、ffmpeg、python3-venv 已安装
- [ ] `tools-asr`、`tools-api` 均为 active
- [ ] `curl` 两个 health 接口正常
- [ ] Nginx 配置 `nginx -t` 通过
- [ ] 百度云安全组已放行 80
- [ ] `.env` 中 `FRONTEND_ORIGIN` 与访问地址一致
