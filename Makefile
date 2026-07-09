# Go 可执行文件（避免 make 继承不到 Homebrew PATH）
ifeq ($(wildcard /opt/homebrew/bin/go),)
  ifeq ($(wildcard /usr/local/go/bin/go),)
    GO := go
  else
    GO := /usr/local/go/bin/go
  endif
else
  GO := /opt/homebrew/bin/go
endif

export PATH := /opt/homebrew/bin:/usr/local/bin:/usr/local/go/bin:$(PATH)
export GOPROXY ?= https://goproxy.cn,direct
export GOSUMDB ?= sum.golang.google.cn

.PHONY: check-go dev install-asr build-api run-api run-asr install-ocr run-ocr tidy

check-go:
	@test -x "$(GO)" || command -v "$(GO)" >/dev/null 2>&1 || { \
		echo "未找到 Go。请先安装："; \
		echo "  macOS:  brew install go"; \
		echo "  Ubuntu: 见 deploy/ubuntu-baidu.md"; \
		exit 1; \
	}
	@"$(GO)" version

dev: check-go
	@"$(GO)" run ./cmd/server

tidy: check-go
	@"$(GO)" mod tidy

build-api: check-go
	@mkdir -p bin
	@"$(GO)" build -o bin/server ./cmd/server

run-api: build-api
	./bin/server

install-asr:
	cd asr-service && python3 -m venv .venv && . .venv/bin/activate && pip install -r requirements.txt

run-asr:
	cd asr-service && . .venv/bin/activate && uvicorn main:app --host 127.0.0.1 --port 18081

install-ocr:
	cd ocr-service && python3 -m venv .venv && . .venv/bin/activate && pip install -r requirements.txt

run-ocr:
	cd ocr-service && . .venv/bin/activate && uvicorn main:app --host 127.0.0.1 --port 18083

# 本地联调：终端1 make run-asr，终端2 make run-ocr，终端3 make dev
