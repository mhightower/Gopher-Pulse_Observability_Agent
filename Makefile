BINARY     := pulse-agent
CMD        := ./cmd/pulse-agent
TEST_FLAGS := -race -count=1
TEST       ?= .

.PHONY: build test coverage lint fmt clean run

build:
	go build -o $(BINARY) $(CMD)

test:
	go test $(TEST_FLAGS) ./... -run $(TEST) -v

coverage:
	go test -cover -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint:
	golangci-lint run

fmt:
	gofmt -w .
	goimports -w .

clean:
	rm -f $(BINARY) coverage.out

run: build
	./$(BINARY) --repo="golang/go" --interval=15s
