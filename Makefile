# 蔚小芯 Makefile
# 用法: make dev | make build | make test | make lint | make clean

APP_NAME   := wxx-server
GO_DIR     := ./server
BUILD_DIR  := ./bin
FLUTTER_DIR:= ./frontend

# ---- Go 后端 ----
# 注：go.mod 已移至项目根目录，所有 go 命令从根目录执行。
# modernc.org/sqlite 纯 Go 驱动已内置 FTS5，无需额外构建标签。
.PHONY: dev build test lint clean migrate

dev:
	go run .

build:
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) ./server/cmd/server

test:
	go test ./... -v -cover

lint:
	go vet ./...

migrate:
	@echo "执行 SQLite 迁移..."
	go run ./server/cmd/migrate

# 同步数据库：将本地 SQLite 的 schema 和数据同步到 Turso
# 使用 .env 中的 TURSO_DB_URL / TURSO_DB_TOKEN
sync-db:
	@echo "同步数据库到 Turso..."
	go run ./server/cmd/sync-db

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
	cd $(FLUTTER_DIR) && flutter build web --release

# ASCII 安全路径构建 — 复制项目到临时 ASCII 目录构建后拷回
FLUTTER_BUILD_TMP := E:/wxx_flutter_tmp
flutter-build-web-safe:
	@echo "=== 步骤 1/4: 清理临时目录 ==="
	rm -rf $(FLUTTER_BUILD_TMP)
	@echo "=== 步骤 2/4: 复制项目到 ASCII 路径 ==="
	cp -r $(FLUTTER_DIR) $(FLUTTER_BUILD_TMP)
	@echo "=== 步骤 3/4: 在安全路径构建 ==="
	cd $(FLUTTER_BUILD_TMP) && flutter build web --release
	@echo "=== 步骤 4/4: 拷贝构建产物回原目录 ==="
	cp -r $(FLUTTER_BUILD_TMP)/build/web $(FLUTTER_DIR)/build/web
	rm -rf $(FLUTTER_BUILD_TMP)
	@echo "=== 构建完成: $(FLUTTER_DIR)/build/web ==="

# 指定输出目录构建 — 直接将产物放到 ASCII 路径
flutter-build-web-output:
	cd $(FLUTTER_DIR) && flutter build web --release --output $(FLUTTER_BUILD_TMP)
	@echo "=== 构建完成: $(FLUTTER_BUILD_TMP) ==="

# 直接构建 APK — 需满足两个前置条件:
# 1. FLUTTER_STORAGE_BASE_URL 环境变量未指向不可用镜像（如 Ohos）
#    若已设置: export FLUTTER_STORAGE_BASE_URL="https://storage.googleapis.com"
# 2. 项目路径不含中文/非 ASCII 字符（impellerc 限制）
#    否则请用 make flutter-build-apk-safe
# 输出后会自动复制为「蔚小芯.apk」（命名规则见 docs/deployment.md）
flutter-build-apk:
	cd $(FLUTTER_DIR) && FLUTTER_STORAGE_BASE_URL="$${FLUTTER_STORAGE_BASE_URL:-https://storage.googleapis.com}" flutter build apk --release
	cp "$(FLUTTER_DIR)/build/app/outputs/apk/release/蔚小芯-release.apk" "$(FLUTTER_DIR)/build/app/outputs/flutter-apk/蔚小芯.apk"
	@echo "=== APK 输出: $(FLUTTER_DIR)/build/app/outputs/flutter-apk/蔚小芯.apk ==="

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
	cp "$(APK_BUILD_TMP)/build/app/outputs/apk/release/蔚小芯-release.apk" "$(FLUTTER_DIR)/build/app/outputs/flutter-apk/蔚小芯.apk"
	rm -rf $(APK_BUILD_TMP)
	@echo "=== APK 构建完成 ==="
	ls -lh $(FLUTTER_DIR)/build/app/outputs/flutter-apk/

flutter-test:
	cd $(FLUTTER_DIR) && flutter test

# ---- Cloudflare Pages 前端部署 ----
.PHONY: deploy-web deploy-web-prebuilt deploy-release

# 标准部署：构建 Flutter Web 后同步 Pages Functions，发布到 wxx-agent 项目
# 域名: https://wxx-agent.pages.dev （详见 docs/蔚小芯前端重新部署.md）
deploy-web: flutter-build-web
	cd $(FLUTTER_DIR)/functions && npm install
	cd $(FLUTTER_DIR) && rm -rf deploy && mkdir -p deploy && cp -rf build/web/* deploy/ && cp -rf functions deploy/ && rm -f deploy/_routes.json
	cd $(FLUTTER_DIR) && npx --yes wrangler pages deploy deploy --project-name wxx-agent --branch main
	@echo "=== 已部署到 https://wxx-agent.pages.dev ==="

# 仅推送已存在的 build/web 产物（不重新编译）
deploy-web-prebuilt:
	cd $(FLUTTER_DIR)/functions && npm install
	cd $(FLUTTER_DIR) && rm -rf deploy && mkdir -p deploy && cp -rf build/web/* deploy/ && cp -rf functions deploy/ && rm -f deploy/_routes.json
	cd $(FLUTTER_DIR) && npx --yes wrangler pages deploy deploy --project-name wxx-agent --branch main
	@echo "=== 已部署到 https://wxx-agent.pages.dev ==="

# 发布 Web + APK：版本号 patch 自动 +1，APK 注入 build/web/downloads 后部署 Cloudflare Pages。
deploy-release:
	pwsh -ExecutionPolicy Bypass -NoProfile -File scripts/build-all.ps1
	cd $(FLUTTER_DIR)/functions && npm install
	cd $(FLUTTER_DIR) && rm -rf deploy && mkdir -p deploy && cp -rf build/web/* deploy/ && cp -rf functions deploy/ && rm -f deploy/_routes.json
	cd $(FLUTTER_DIR) && npx --yes wrangler pages deploy deploy --project-name wxx-agent --branch main
	@echo "=== 已发布 Web + APK 到 https://wxx-agent.pages.dev ==="

# ---- 前端全量构建 ----
# 顺序构建 Web + APK（调用 PowerShell 7 脚本）
.PHONY: all-frontend

all-frontend:
	pwsh -ExecutionPolicy Bypass -NoProfile -File scripts/build-all.ps1

# ---- 全部 ----
.PHONY: all test-all hooks all-safe all-apk-safe

all: build flutter-build-web

# 全量构建（使用 ASCII 安全路径，适用于项目路径含中文的环境）
all-safe: build flutter-build-web-output

all-apk-safe: flutter-build-apk-safe

test-all: test flutter-test

# ---- 评测与压测 ----
.PHONY: test-eval stress quality-gate

test-eval:
	@echo "运行问答质量评测（需要 -token 参数）..."
	cd server && go run -tags fts5 cmd/eval/main.go -baseline ../specs/eval-baseline.ndjson -token $(TOKEN)

quality-gate:
	@echo "运行质量门禁检查..."
	go run ./server/cmd/gate -report $(or $(REPORT),eval-result.json)

stress:
	@echo "运行压测（默认 50 并发）..."
	cd server && go run -tags fts5 cmd/stress/main.go -c $(or $(CONCURRENCY),50) -token $(TOKEN)

hooks:
	@echo "安装 pre-commit 钩子..."
	cp scripts/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	@echo "pre-commit 钩子安装完成"

# ---- Temporal 工作流引擎 ----
.PHONY: temporal-dev temporal-install

temporal-dev:
	@echo "启动 Temporal 开发服务器..."
	temporal server start-dev --ip 0.0.0.0 --port 7233

temporal-install:
	@echo "请从以下地址下载 Temporal CLI 二进制："
	@echo "  https://github.com/temporalio/cli/releases"
	@echo "或使用包管理器安装："
	@echo "  macOS:  brew install temporal"
	@echo "  Linux:  curl -sSf https://temporal.download/cli.sh | sh"
	@echo "  Windows: scoop bucket add temporal https://github.com/temporalio/scoop-temporal.git && scoop install temporal"
