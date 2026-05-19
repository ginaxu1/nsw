.PHONY: help install clean dev build lint format type-check preview

help:
	@echo "📦 NSW OGA Officer Portal - Development Commands"
	@echo ""
	@echo "Setup & Installation:"
	@echo "  make install        - Install all dependencies"
	@echo "  make clean          - Remove node_modules and dist files"
	@echo ""
	@echo "Development:"
	@echo "  make dev            - Run OGA app in dev mode"
	@echo ""
	@echo "Build:"
	@echo "  make build          - Build OGA app"
	@echo ""
	@echo "Quality:"
	@echo "  make lint           - Run ESLint"
	@echo "  make format         - Auto-fix lint and formatting issues"
	@echo "  make type-check     - Run TypeScript type checking"
	@echo ""

install:
	@echo "📦 Installing dependencies with pnpm..."
	@pnpm install

clean:
	@echo "🧹 Cleaning workspace..."
	@rm -rf node_modules app/node_modules
	@rm -rf app/dist
	@echo "✅ Cleaned!"

dev:
	@echo "🚀 Starting OGA app..."
	@pnpm --filter oga-app dev

build:
	@echo "🏗️  Building OGA app..."
	@pnpm --filter oga-app build

lint:
	@echo "🔍 Running ESLint..."
	@pnpm --filter oga-app lint

format:
	@echo "🔧 Auto-fixing lint and formatting issues..."
	@pnpm --filter oga-app format
	@echo "✅ Formatting complete!"

type-check:
	@echo "📝 Running TypeScript type checking..."
	@pnpm --filter oga-app type-check

preview:
	@echo "👀 Previewing OGA app..."
	@pnpm --filter oga-app preview
