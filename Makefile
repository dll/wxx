# 蔚小芯 Makefile
# 用法: make dev | make build | make test | make lint | make clean

APP_NAME   := wxx-server
GO_DIR     := ./server
BUILD_DIR  := ./bin
FLUTTER_DIR:= ./frontend

# ---- Go 后端 ----
.PHONY: dev build test lint clean migrate

dev:
	cd $(GO_DIR) && go run ./cmd/server

build:
	cd $(GO_DIR) && go build -o ../$(BUILD_DIR)/$(APP_NAME) ./cmd/server

test:
	cd $(GO_DIR) && go test ./... -v -cover

lint:
	cd $(GO_DIR) && go vet ./...

migrate:
	@echo "执行 SQLite 迁移..."
	cd $(GO_DIR) && go run ./cmd/migrate

clean:
	rm -rf $(BUILD_DIR)

# ---- Flutter 前端 ----
.PHONY: flutter-get flutter-run flutter-build flutter-test

flutter-get:
	cd $(FLUTTER_DIR) && flutter pub get

flutter-run:
	cd $(FLUTTER_DIR) && flutter run

flutter-build-web:
	cd $(FLUTTER_DIR) && flutter build web

flutter-build-apk:
	cd $(FLUTTER_DIR) && flutter build apk --release

flutter-test:
	cd $(FLUTTER_DIR) && flutter test

# ---- 全部 ----
.PHONY: all test-all

all: build flutter-build-web

test-all: test flutter-test
