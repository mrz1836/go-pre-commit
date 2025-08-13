.PHONY: build clean test

build:
	@echo "Building license-checker..."
	@go build -o license-checker main.go
	@chmod +x license-checker

clean:
	@rm -f license-checker

test:
	@echo '{"command":"check","files":["main.go"]}' | go run main.go
