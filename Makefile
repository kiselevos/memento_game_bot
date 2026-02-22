# ============================================================
# 🧠 Default & Help
# ============================================================
default: help

.PHONY: help
help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# ============================================================
# ⚙️ Development Commands
# ============================================================

.PHONY: tidy
tidy: ## Обновление зависимостей проекта
	@echo "🔍 Installing app packages..."
	@go mod tidy
	@go mod download
	@echo "✅ Packages installed"

.PHONY: run
run: ## Запуск бота локально (Подключение к DB в docker)
	@go run ./cmd/main.go

DB_COMPOSE=docker-compose.db.yml
BOT_COMPOSE=docker-compose.yml

.PHONY: network
network: ## Создать общую docker сеть (если нет)
	@docker network inspect memento_network >/dev/null 2>&1 || docker network create memento_network
	@echo "🌐 Network ready"


# ============================================================
# DATABASE
# ============================================================

.PHONY: db-up
db-up: network ## Запуск PostgreSQL + миграции
	@docker compose -f $(DB_COMPOSE) up -d
	@echo "✅ PostgreSQL + migrations started"


.PHONY: db-stop
db-stop: ## Остановка PostgreSQL без удаления данных
	@docker compose -f $(DB_COMPOSE) stop
	@echo "⏸ PostgreSQL stopped"

.PHONY: db-down
db-down: ## Остановка и удаление PostgreSQL + данных
	@docker compose -f $(DB_COMPOSE) down -v
	@echo "🧹 PostgreSQL removed (data deleted)"


.PHONY: migrate
migrate: ## Прогнать миграции вручную
	@docker compose -f $(DB_COMPOSE) run --rm migrate
	@echo "✅ Migrations applied"

.PHONY: logs-db
logs-db: ## Логи Postgres
	@docker compose -f $(DB_COMPOSE) logs -f postgres


# =========================
# BOT
# =========================

.PHONY: bot-up
bot-up: network ## Запуск бота в Docker
	@docker compose -f $(BOT_COMPOSE) up -d --build
	@echo "🤖 Bot started"

.PHONY: bot-stop
bot-stop: ## Остановка бота
	@docker compose -f $(BOT_COMPOSE) stop
	@echo "⏸ Bot stopped"

.PHONY: bot-down
bot-down: ## Удаление контейнера бота
	@docker compose -f $(BOT_COMPOSE) down
	@echo "🧹 Bot removed"

.PHONY: logs-bot
logs-bot: ## Логи бота
	@docker compose -f $(BOT_COMPOSE) logs -f bot

# =========================
# FULL STACK
# =========================

.PHONY: up
up: db-up bot-up ## Полный запуск (DB + Bot)
	@echo "🚀 Full stack started"

.PHONY: stop
stop: bot-stop db-stop ## Остановить всё
	@echo "🛑 Full stack stopped"

.PHONY: down
down: bot-down db-down ## Удалить всё (включая данные БД)
	@echo "🧹 Full stack removed"

.PHONY: ps
ps: ## Показать контейнеры проекта
	@docker ps --filter "network=memento_network"


# ============================================================
# 🧪 Tests & Checks
# ============================================================
.PHONY: test-all
test-all: ## Run all tests include postgres-data
	@go test ./... -v
	@echo "✅ Tests finished"


PHONY: lint
lint: ## Run fmt & vet
	@go fmt ./...
	@go vet ./...
	@echo "✅ Go vet & fmt success complete"
