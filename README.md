# Tools Web Backend

工具网站后端：**Go API** + **Python ASR 服务**（Whisper）。

> 部署方式：**Ubuntu 原生 + systemd + Nginx**，不使用 Docker。

## 架构

```
Frontend → Go API (127.0.0.1:18080) → ffmpeg / yt-dlp
                          → Python ASR (127.0.0.1:18081, faster-whisper)
                          → 百度云 BOS（可选）
```

## 目录

```
tools-web-backend/
├── cmd/server/          # Go API 入口
├── internal/            # Go 业务代码
├── asr-service/         # Python 语音识别服务
├── deploy/              # Ubuntu 部署（systemd + Nginx）
├── Makefile             # 本地启动命令
└── .env.example
```

## 本地开发

### 1. 系统依赖（Ubuntu）

```bash
sudo apt update
sudo apt install -y ffmpeg python3 python3-venv python3-pip
pip3 install yt-dlp --break-system-packages   # 或 python3 -m pip install --user yt-dlp
```

### 2. 配置环境

```bash
cp .env.example .env
# 按需修改 FRONTEND_ORIGIN、BOS 等
```

### 3. 启动 ASR 服务（终端 1）

```bash
make install-asr    # 首次
make run-asr
```

有 GPU 时，启动前：

```bash
export WHISPER_DEVICE=cuda
export WHISPER_COMPUTE_TYPE=float16
export WHISPER_MODEL=large-v3
make run-asr
```

### 4. 启动 Go API（终端 2）

```bash
go mod tidy
make dev
# 或: make run-api
```

## 生产部署（百度云 Ubuntu）

完整步骤见 **[deploy/ubuntu-baidu.md](deploy/ubuntu-baidu.md)**，概要：

1. 安装 Go、Node、ffmpeg、yt-dlp、Nginx  
2. `asr-service` 用 **systemd** 托管（`tools-asr.service`）  
3. Go API 编译后用 **systemd** 托管（`tools-api.service`）  
4. 前端 `npm run build`，Nginx 托管静态文件并反代 `/api`

## API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/health | 健康检查 |
| POST | /api/v1/tasks/media-to-text | 提交视频/音频 URL |
| POST | /api/v1/tasks/media-to-text/upload | 上传文件 |
| GET | /api/v1/tasks/:id | 查询任务 |
| GET | /api/v1/tasks | 任务列表 |

### 提交 URL

```json
POST /api/v1/tasks/media-to-text
{
  "source_url": "https://example.com/video.mp4",
  "language": "zh"
}
```

### 上传文件

```bash
curl -F "file=@demo.mp4" -F "language=zh" \
  http://localhost:18080/api/v1/tasks/media-to-text/upload
```

## 百度云 BOS

`.env` 中开启：

```env
BOS_ENABLED=true
BOS_ENDPOINT=https://bj.bcebos.com
BOS_BUCKET=your-bucket
BOS_ACCESS_KEY=xxx
BOS_SECRET_KEY=xxx
```

## 常用运维命令

```bash
# 查看服务状态
sudo systemctl status tools-asr tools-api nginx

# 重启
sudo systemctl restart tools-asr tools-api

# 看日志
sudo journalctl -u tools-api -f
sudo journalctl -u tools-asr -f
```
