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

.PHONY: db-up
db-up: ## Запуск PostgreSQL контейнера
	@docker compose -f docker-compose.db.yml up -d postgres
	@echo "✅ PostgreSQL created & run"

.PHONY: db-stop
db-stop: ## Остановка PostgreSQL контейнера без удаления данных
	@docker compose -f docker-compose.db.yml stop postgres
	@echo "⏸ PostgreSQL stopped"

.PHONY: db-down
db-down: ## Остановка и удаление PostgreSQL контейнера и данных
	@docker compose -f docker-compose.db.yml down -v
	@echo "🧹 PostgreSQL stopped & removed"

.PHONY: migrate
migrate: ## Запуск миграций в Docker
	@docker compose -f docker-compose.db.yml run --rm migrate
	@echo "✅ Migrate success"

.PHONY: rebuild
rebuild: ## Персборка образа для миграций
	@docker compose -f docker-compose.db.yml build migrate

.PHONY: logs-db
logs-db: ## Логи PostgreSQL контейнера
	@docker logs -f pbb_postgres

.PHONY: clean
clean: ## Остановка и удаление всех Postgres контенеров, данных DB 
	@docker compose -f docker-compose.db.yml down -v --remove-orphans

.PHONY: ps
ps: ## Показать запушенные контенеры Docker
	@docker ps --filter "name=pbb_postgres"

# ============================================================
# 🧩 Combined Shortcuts
# ============================================================

.PHONY: setup
setup: db-up migrate ## Поднятие базы и установка миграций

.PHONY: restart
restart: db-down db-up migrate ## Перезапуск DB и миграций