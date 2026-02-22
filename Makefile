APP := syl-wordcount
GO ?= go
BIN_DIR ?= bin
BIN := $(BIN_DIR)/$(APP)
DESTDIR ?=
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X 'syl-wordcount/cmd.Version=$(VERSION)' -X 'syl-wordcount/cmd.Commit=$(COMMIT)' -X 'syl-wordcount/cmd.BuildTime=$(BUILD_TIME)'
GO_BIN_DIR ?= $(shell sh -c 'gobin="$$( $(GO) env GOBIN )"; if [ -n "$$gobin" ]; then printf "%s" "$$gobin"; else gopath="$$( $(GO) env GOPATH )"; printf "%s/bin" "$${gopath%%:*}"; fi')
INSTALL_BIN_DIR := $(DESTDIR)$(GO_BIN_DIR)
INSTALL_BIN := $(INSTALL_BIN_DIR)/$(APP)

.DEFAULT_GOAL := default

.PHONY: default help build test fmt tidy clean install uninstall

default:
	@$(MAKE) fmt
	@$(MAKE) test
	@$(MAKE) install

help:
	@echo "Targets:"
	@echo "  make            - 默认流程：fmt -> test -> install"
	@echo "  make build      - 编译二进制到 $(BIN)"
	@echo "  make test       - 运行全部测试"
	@echo "  make fmt        - gofmt 全部 Go 文件"
	@echo "  make tidy       - 整理 go.mod/go.sum"
	@echo "  make install    - 安装到 Go bin 目录"
	@echo "  make uninstall  - 卸载已安装二进制"
	@echo "  make clean      - 删除构建产物"

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN) .

test:
	$(GO) test ./...

fmt:
	@gofmt -w $$(find . -name '*.go' -type f)

tidy:
	$(GO) mod tidy

clean:
	rm -rf $(BIN_DIR)

install: build
	@mkdir -p "$(INSTALL_BIN_DIR)"
	install -m 0755 "$(BIN)" "$(INSTALL_BIN)"

uninstall:
	rm -f "$(INSTALL_BIN)"
