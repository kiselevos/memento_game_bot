# ============================================================
# 🧠 Default & Help
# ============================================================
default: help

.PHONY: help
help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# ============================================================
# ⚙️ Setup Tools
# ============================================================
.PHONY: run
run: 
	@go run ./cmd/main.go


.PHONY: db-run
db-run: 
	@docker compose -f docker-compose.db.yml up -d

