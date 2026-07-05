.PHONY: build run clean test

build:
	go build -o dist/byok .

run:
	go run main.go $(ARGS)

clean:
	cmd /c "if exist dist rmdir /s /q dist"

test:
	go test ./...