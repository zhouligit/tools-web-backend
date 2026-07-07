# 确保 Homebrew / 官方 Go 在 PATH 中（make 有时继承不到完整 PATH）
export PATH := /opt/homebrew/bin:/usr/local/bin:/usr/local/go/bin:$(PATH)

GO ?= go

.PHONY: check-go dev run build-asr install-asr build-api run-asr run-api

check-go:
	@command -v $(GO) >/dev/null 2>&1 || { \
		echo "未找到 Go。请先安装："; \
		echo "  macOS:  brew install go"; \
		echo "  Ubuntu: 见 deploy/ubuntu-baidu.md"; \
		exit 1; \
	}
	@$(GO) version

dev: check-go
	$(GO) run ./cmd/server

build-api: check-go
	@mkdir -p bin
	$(GO) build -o bin/server ./cmd/server

run-api: build-api
	./bin/server

install-asr:
	cd asr-service && python3 -m venv .venv && . .venv/bin/activate && pip install -r requirements.txt

run-asr:
	cd asr-service && . .venv/bin/activate && uvicorn main:app --host 127.0.0.1 --port 8090

# 本地联调：开两个终端分别 make run-asr / make dev
