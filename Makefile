# ==============================================================================
#  S&P 500 TRACKER - MASTER MAKEFILE
#  Env: Linux (Chromebook) -> Target: Windows (.exe)
# ==============================================================================

BINARY_NAME=sp500-tracker
FRONTEND_DIR=frontend

# Detect Operating System
OS := $(shell uname -s)

.PHONY: all setup deps dev build-windows clean help

# Default target: Show help
all: help

# ------------------------------------------------------------------------------
# 🛠️ SETUP & DEPENDENCIES
# ------------------------------------------------------------------------------

# 1. Install System Tools (Chromebook/Debian specific)
# We need 'mingw-w64' to compile Windows C++ bindings on Linux
setup:
	@echo "🔧 Installing system build tools..."
	sudo apt-get update
	sudo apt-get install -y build-essential mingw-w64

# 2. Install Project Dependencies (Go & Node)
deps:
	@echo "📦 Installing Go dependencies..."
	go get github.com/wailsapp/wails/v2
	go get github.com/gorilla/websocket
	go get github.com/piquette/finance-go
	go get github.com/PuerkitoBio/goquery
	go mod tidy
	@echo "📦 Installing Frontend dependencies..."
	cd $(FRONTEND_DIR) && npm install

# ------------------------------------------------------------------------------
# 🚀 DEVELOPMENT
# ------------------------------------------------------------------------------

# Run the app in "Live Reload" mode on your machine
dev:
	@echo "👀 Starting Wails Dev Mode..."
	wails dev

# ------------------------------------------------------------------------------
# 📦 PRODUCTION BUILD (The Deliverable)
# ------------------------------------------------------------------------------

# Build the Windows .exe (Cross-Compile)
build-windows:
	@echo "🔨 Building Windows Binary (amd64)..."
	wails build -platform windows/amd64 -o $(BINARY_NAME).exe
	@echo "✅ SUCCESS!"
	@echo "📂 Your file is ready at: build/bin/$(BINARY_NAME).exe"

# ------------------------------------------------------------------------------
# 🧹 UTILITIES
# ------------------------------------------------------------------------------

clean:
	@echo "🧹 Cleaning up artifacts..."
	rm -rf build/bin/*
	rm -rf $(FRONTEND_DIR)/dist

help:
	@echo "Available commands:"
	@echo "  make setup         -> Install MinGW (Required for Windows build on Linux)"
	@echo "  make deps          -> Install Go & NPM packages"
	@echo "  make dev           -> Run locally for testing"
	@echo "  make build-windows -> Generate the final .exe for the client"
	@echo "  make clean         -> Delete build folder"