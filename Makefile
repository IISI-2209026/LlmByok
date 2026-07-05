.PHONY: build run clean test

# 版本號：從 internal/version/version.go 讀取，可用 VERSION 環境變數覆寫。
VERSION ?= $(shell sed -n 's/.*Version = "\([^"]*\)".*/\1/p' internal/version/version.go)
LDFLAGS := -X github.com/IISI-2209026/LlmByok/internal/version.Version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o dist/byok .

run:
	go run main.go $(ARGS)

clean:
	cmd /c "if exist dist rmdir /s /q dist"

test:
	go test ./...