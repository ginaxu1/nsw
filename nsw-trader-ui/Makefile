.PHONY: help install clean dev build lint format type-check preview

help:
	@echo "📦 NSW Trader Portal - Development Commands"
	@echo ""
	@echo "Setup & Installation:"
	@echo "  make install        - Install all dependencies"
	@echo "  make clean          - Remove node_modules and dist files"
	@echo ""
	@echo "Development:"
	@echo "  make dev            - Run Trader app in dev mode"
	@echo ""
	@echo "Build:"
	@echo "  make build          - Build all workspaces"
	@echo ""
	@echo "Quality:"
	@echo "  make lint           - Run ESLint on all workspaces"
	@echo "  make format         - Auto-fix lint and formatting issues"
	@echo "  make type-check     - Run TypeScript type checking"
	@echo ""

install:
	@echo "📦 Installing dependencies with pnpm..."
	@pnpm install

clean:
	@echo "🧹 Cleaning workspace..."
	@rm -rf node_modules app/node_modules packages/*/node_modules
	@rm -rf packages/*/dist app/dist
	@echo "✅ Cleaned!"

dev:
	@echo "🚀 Starting Trader app..."
	@pnpm --filter trader-app dev

build:
	@echo "🏗️  Building all workspaces..."
	@pnpm -r build

lint:
	@echo "🔍 Running ESLint..."
	@pnpm -r lint

format:
	@echo "🔧 Auto-fixing lint and formatting issues..."
	@pnpm -r run format
	@echo "✅ Formatting complete!"

type-check:
	@echo "📝 Running TypeScript type checking..."
	@pnpm run type-check

preview:
	@echo "👀 Previewing Trader app..."
	@pnpm --filter trader-app preview
