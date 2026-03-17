BINARY     := pulse-agent
CMD        := ./cmd/pulse-agent
TEST_FLAGS := -race -count=1
TEST       ?= .

.PHONY: build test coverage lint fmt clean run stop \
        stack-up stack-down stack-logs \
        docker-build stack-full-up stack-full-down

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

stop:
	@lsof -ti :9464 | xargs kill 2>/dev/null && echo "pulse-agent stopped" || echo "pulse-agent is not running"

stack-up:
	docker compose up -d
	@echo "Grafana: http://localhost:3000  (admin/admin)"
	@echo "Prometheus: http://localhost:9090"

stack-down:
	docker compose down

stack-logs:
	docker compose logs -f

docker-build:
	docker build -t pulse-agent:latest .

stack-full-up: docker-build
	docker compose -f docker-compose.full.yml up -d
	@echo "Grafana: http://localhost:3000  (admin/admin)"
	@echo "Prometheus: http://localhost:9090"
	@echo "Agent metrics: http://localhost:9464/metrics"
	@echo "Agent health:  http://localhost:9464/health"

stack-full-down:
	docker compose -f docker-compose.full.yml down
