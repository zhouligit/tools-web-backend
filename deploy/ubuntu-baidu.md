# 百度云 Ubuntu 部署指南（无 Docker）

在 Ubuntu 上直接安装依赖，用 **systemd** 管理进程，**Nginx** 对外提供服务。

## 0. 目录规划

| 路径 | 用途 |
|------|------|
| `/opt/tools-web/tools-web-backend` | 后端代码 |
| `/opt/tools-web/tools-web-frontend` | 前端代码 |
| `/opt/tools-web/bin/server` | Go 编译产物 |
| `/var/lib/tools-web/tmp` | 临时音视频（可在 .env 配置） |

```bash
sudo mkdir -p /opt/tools-web/bin /var/lib/tools-web/tmp
sudo chown -R $USER:$USER /opt/tools-web /var/lib/tools-web
```

## 1. 安装系统依赖

```bash
sudo apt update && sudo apt upgrade -y
sudo apt install -y git curl wget nginx ffmpeg \
  python3 python3-venv python3-pip build-essential

pip3 install yt-dlp --break-system-packages
```

## 2. 安装 Go

```bash
wget https://go.dev/dl/go1.22.5.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version
```

## 3. 安装 Node.js（前端构建）

```bash
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs
node -v && npm -v
```

## 4. 拉取代码

```bash
cd /opt/tools-web
git clone <your-backend-repo> tools-web-backend
git clone <your-frontend-repo> tools-web-frontend
```

## 5. 配置环境变量

```bash
cd /opt/tools-web/tools-web-backend
cp .env.example .env
vim .env
```

示例：

```env
ADDR=:8080
FRONTEND_ORIGIN=https://tools.example.com
TEMP_DIR=/var/lib/tools-web/tmp
MAX_UPLOAD_MB=500
ASR_SERVICE_URL=http://127.0.0.1:8090

BOS_ENABLED=true
BOS_ENDPOINT=https://bj.bcebos.com
BOS_BUCKET=your-bucket-name
BOS_ACCESS_KEY=your-ak
BOS_SECRET_KEY=your-sk
```

## 6. 部署 ASR 服务（systemd）

```bash
cd /opt/tools-web/tools-web-backend/asr-service
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
deactivate
```

创建 systemd 单元（**把 User=ubuntu 改成你的用户名**）：

```bash
sudo tee /etc/systemd/system/tools-asr.service <<'EOF'
[Unit]
Description=Tools Web ASR Service (faster-whisper)
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/opt/tools-web/tools-web-backend/asr-service
Environment=WHISPER_MODEL=medium
Environment=WHISPER_DEVICE=cpu
Environment=WHISPER_COMPUTE_TYPE=int8
ExecStart=/opt/tools-web/tools-web-backend/asr-service/.venv/bin/uvicorn main:app --host 127.0.0.1 --port 8090
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable tools-asr
sudo systemctl start tools-asr
sudo systemctl status tools-asr
```

GPU 实例修改 Environment：

```ini
Environment=WHISPER_DEVICE=cuda
Environment=WHISPER_COMPUTE_TYPE=float16
Environment=WHISPER_MODEL=large-v3
```

## 7. 编译并部署 Go API（systemd）

```bash
cd /opt/tools-web/tools-web-backend
go mod tidy
go build -o /opt/tools-web/bin/server ./cmd/server
```

```bash
sudo tee /etc/systemd/system/tools-api.service <<'EOF'
[Unit]
Description=Tools Web Go API
After=network.target tools-asr.service
Requires=tools-asr.service

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/opt/tools-web/tools-web-backend
EnvironmentFile=/opt/tools-web/tools-web-backend/.env
ExecStart=/opt/tools-web/bin/server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable tools-api
sudo systemctl start tools-api
sudo systemctl status tools-api
```

## 8. 构建前端 + Nginx

```bash
cd /opt/tools-web/tools-web-frontend
npm ci
npm run build
```

Nginx 配置（**修改 server_name**）：

```bash
sudo tee /etc/nginx/sites-available/tools-web <<'EOF'
server {
    listen 80;
    server_name tools.example.com;

    root /opt/tools-web/tools-web-frontend/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
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

sudo ln -sf /etc/nginx/sites-available/tools-web /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t && sudo systemctl reload nginx
```

HTTPS：使用 certbot 或百度云 SSL 证书，在 server 块增加 `listen 443 ssl` 配置。

## 9. 验证

```bash
curl -s http://127.0.0.1:8090/health | head
curl -s http://127.0.0.1:8080/api/v1/health | head
curl -I http://127.0.0.1/
```

浏览器访问 `http://你的域名`，测试「音视频转文字」。

## 10. 升级发布

```bash
# 后端
cd /opt/tools-web/tools-web-backend && git pull
go build -o /opt/tools-web/bin/server ./cmd/server
sudo systemctl restart tools-api

# ASR（若 requirements 有变）
cd asr-service && source .venv/bin/activate && pip install -r requirements.txt
sudo systemctl restart tools-asr

# 前端
cd /opt/tools-web/tools-web-frontend && git pull && npm ci && npm run build
sudo systemctl reload nginx
```

## 11. 故障排查

| 现象 | 排查 |
|------|------|
| API 502 | `systemctl status tools-api`，`journalctl -u tools-api -n 50` |
| 转写失败 | `journalctl -u tools-asr -f`，确认模型下载完成 |
| 上传超时 | Nginx `client_max_body_size`、`proxy_read_timeout` |
| URL 下载失败 | 服务器执行 `yt-dlp --version`，检查目标站是否可访问 |
| ffmpeg 报错 | `ffmpeg -version`，确认已安装 |

## 12. 资源与费用

| 场景 | CPU | 内存 | GPU |
|------|-----|------|-----|
| 轻量 medium | 4 核 | 8 GB | 无 |
| 推荐 | 8 核 | 16 GB | 可选 T4 |
| 高准确 large-v3 | 8 核+ | 16 GB+ | 建议 |

- 无 Docker、无云 ASR API 费用  
- BOS 按存储/流量计费（可选）
