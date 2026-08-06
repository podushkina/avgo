SHELL := /bin/bash
COMPOSE := docker compose

.DEFAULT_GOAL := help

.PHONY: help
help: ## Показать список команд
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

.PHONY: env
env: ## Создать .env из .env.example, если его ещё нет
	@test -f .env || (cp .env.example .env && echo "создан .env")

.PHONY: up
up: env ## Поднять стек (LLM — нативный Ollama на хосте, быстро на macOS)
	$(COMPOSE) up -d --build

.PHONY: up-llm
up-llm: env ## Поднять стек вместе с Ollama в контейнере (Linux/деплой/жюри)
	LLM_BASE_URL=http://ollama:11434/v1 $(COMPOSE) --profile local-llm up -d --build

.PHONY: down
down: ## Остановить стек
	$(COMPOSE) --profile local-llm down

.PHONY: clean
clean: ## Остановить стек и удалить тома (БД будет пересоздана и пересидена)
	$(COMPOSE) --profile local-llm down -v

.PHONY: logs
logs: ## Хвост логов всех сервисов
	$(COMPOSE) logs -f --tail=100

.PHONY: test
test: ## Go-тесты обоих сервисов
	cd services/api && go test ./...
	cd services/ai && go test ./...

.PHONY: lint
lint: ## golangci-lint по обоим сервисам
	cd services/api && golangci-lint run ./...
	cd services/ai && golangci-lint run ./...

.PHONY: fmt
fmt: ## Форматирование Go-кода
	cd services/api && go fmt ./...
	cd services/ai && go fmt ./...

.PHONY: bench-llm
bench-llm: ## Сравнить модели-кандидаты на промпте мошенника (русский, скорость и качество)
	@python3 scripts/bench_llm.py $(MODELS)

.PHONY: smoke
smoke: ## Сквозная проверка API поднятого стека
	@./scripts/smoke.sh
