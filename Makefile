.PHONY: dev run build-asr install-asr build-api run-asr run-api

dev:
	go run ./cmd/server

build-api:
	go build -o bin/server ./cmd/server

run-api: build-api
	./bin/server

install-asr:
	cd asr-service && python3 -m venv .venv && . .venv/bin/activate && pip install -r requirements.txt

run-asr:
	cd asr-service && . .venv/bin/activate && uvicorn main:app --host 127.0.0.1 --port 8090

# 本地联调：开两个终端分别 make run-asr / make dev
