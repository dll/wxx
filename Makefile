# 蔚小芯 Makefile
# 用法: make dev | make build | make test | make lint | make clean

APP_NAME   := wxx-server
GO_DIR     := ./server
BUILD_DIR  := ./bin
FLUTTER_DIR:= ./frontend
GO_FLAGS   := -tags "fts5"  # 启用 SQLite FTS5 全文检索支持

# ---- Go 后端 ----
.PHONY: dev build test lint clean migrate

dev:
	cd $(GO_DIR) && go run $(GO_FLAGS) ./cmd/server

build:
	cd $(GO_DIR) && go build $(GO_FLAGS) -o ../$(BUILD_DIR)/$(APP_NAME) ./cmd/server

test:
	cd $(GO_DIR) && go test $(GO_FLAGS) ./... -v -cover

lint:
	cd $(GO_DIR) && go vet $(GO_FLAGS) ./...

migrate:
	@echo "执行 SQLite 迁移..."
	cd $(GO_DIR) && go run $(GO_FLAGS) ./cmd/migrate

clean:
	rm -rf $(BUILD_DIR)

# ---- Flutter 前端 ----
.PHONY: flutter-get flutter-run flutter-build-web flutter-build-web-safe flutter-build-web-output flutter-build-apk flutter-build-apk-safe flutter-test

flutter-get:
	cd $(FLUTTER_DIR) && flutter pub get

flutter-run:
	cd $(FLUTTER_DIR) && flutter run

# 默认构建 — 若项目路径含中文（如"学工"），Flutter SDK 3.35 impellerc
# 无法编译 shader，请改用 make flutter-build-web-safe 或 flutter-build-web-output
flutter-build-web:
	cd $(FLUTTER_DIR) && flutter build web

# ASCII 安全路径构建 — 复制项目到临时 ASCII 目录构建后拷回
FLUTTER_BUILD_TMP := E:/wxx_flutter_tmp
flutter-build-web-safe:
	@echo "=== 步骤 1/4: 清理临时目录 ==="
	rm -rf $(FLUTTER_BUILD_TMP)
	@echo "=== 步骤 2/4: 复制项目到 ASCII 路径 ==="
	cp -r $(FLUTTER_DIR) $(FLUTTER_BUILD_TMP)
	@echo "=== 步骤 3/4: 在安全路径构建 ==="
	cd $(FLUTTER_BUILD_TMP) && flutter build web
	@echo "=== 步骤 4/4: 拷贝构建产物回原目录 ==="
	cp -r $(FLUTTER_BUILD_TMP)/build/web $(FLUTTER_DIR)/build/web
	rm -rf $(FLUTTER_BUILD_TMP)
	@echo "=== 构建完成: $(FLUTTER_DIR)/build/web ==="

# 指定输出目录构建 — 直接将产物放到 ASCII 路径
flutter-build-web-output:
	cd $(FLUTTER_DIR) && flutter build web --output $(FLUTTER_BUILD_TMP)
	@echo "=== 构建完成: $(FLUTTER_BUILD_TMP) ==="

# 直接构建 APK — 需满足两个前置条件:
# 1. FLUTTER_STORAGE_BASE_URL 环境变量未指向不可用镜像（如 Ohos）
#    若已设置: export FLUTTER_STORAGE_BASE_URL="https://storage.googleapis.com"
# 2. 项目路径不含中文/非 ASCII 字符（impellerc 限制）
#    否则请用 make flutter-build-apk-safe
flutter-build-apk:
	cd $(FLUTTER_DIR) && FLUTTER_STORAGE_BASE_URL="$${FLUTTER_STORAGE_BASE_URL:-https://storage.googleapis.com}" flutter build apk --release

# ASCII 安全路径构建 APK — 复制到临时 ASCII 目录，覆盖镜像变量
APK_BUILD_TMP := E:/wxx_apk_build
flutter-build-apk-safe:
	@echo "=== 步骤 1/5: 清理临时目录 ==="
	rm -rf $(APK_BUILD_TMP)
	@echo "=== 步骤 2/5: 复制项目到 ASCII 路径 ==="
	cp -r $(FLUTTER_DIR) $(APK_BUILD_TMP)
	@echo "=== 步骤 3/5: 构建 Debug APK ==="
	cd $(APK_BUILD_TMP) && FLUTTER_STORAGE_BASE_URL="https://storage.googleapis.com" flutter build apk --debug
	@echo "=== 步骤 4/5: 构建 Release APK ==="
	cd $(APK_BUILD_TMP) && FLUTTER_STORAGE_BASE_URL="https://storage.googleapis.com" flutter build apk --release
	@echo "=== 步骤 5/5: 拷贝构建产物回原目录 ==="
	cp $(APK_BUILD_TMP)/build/app/outputs/flutter-apk/app-debug.apk $(FLUTTER_DIR)/build/app/outputs/flutter-apk/
	cp $(APK_BUILD_TMP)/build/app/outputs/flutter-apk/app-release.apk $(FLUTTER_DIR)/build/app/outputs/flutter-apk/
	rm -rf $(APK_BUILD_TMP)
	@echo "=== APK 构建完成 ==="
	ls -lh $(FLUTTER_DIR)/build/app/outputs/flutter-apk/

flutter-test:
	cd $(FLUTTER_DIR) && flutter test

# ---- 全部 ----
.PHONY: all test-all hooks all-safe all-apk-safe

all: build flutter-build-web

# 全量构建（使用 ASCII 安全路径，适用于项目路径含中文的环境）
all-safe: build flutter-build-web-output

all-apk-safe: flutter-build-apk-safe

test-all: test flutter-test

hooks:
	@echo "安装 pre-commit 钩子..."
	cp scripts/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	@echo "pre-commit 钩子安装完成"
