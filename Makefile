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
up: env ## Поднять всё в Docker одной командой, включая модель
	$(COMPOSE) up -d --build

.PHONY: up-host-llm
up-host-llm: env ## То же, но модель берётся с хоста (быстрее на macOS, нужен запущенный ollama serve)
	LLM_BASE_URL=http://host.docker.internal:11434/v1 $(COMPOSE) up -d --build

.PHONY: down
down: ## Остановить стек
	$(COMPOSE) down

.PHONY: clean
clean: ## Остановить стек и удалить тома (БД пересоздастся, модель скачается заново)
	$(COMPOSE) down -v

.PHONY: logs
logs: ## Хвост логов всех сервисов
	$(COMPOSE) logs -f --tail=100

.PHONY: test
test: ## Go-тесты с детектором гонок
	cd services/api && go test -race ./...

.PHONY: lint
lint: ## golangci-lint
	cd services/api && golangci-lint run ./...

.PHONY: fmt
fmt: ## Форматирование Go-кода
	cd services/api && go fmt ./...

.PHONY: bench-llm
bench-llm: ## Сравнить модели-кандидаты на промпте мошенника (русский, скорость и качество)
	@python3 scripts/bench_llm.py $(MODELS)

.PHONY: smoke
smoke: ## Сквозная проверка API поднятого стека
	@bash ./scripts/smoke.sh

.PHONY: front-lint
front-lint: ## ESLint фронтенда
	cd frontend && yarn lint
