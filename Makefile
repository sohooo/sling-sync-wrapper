APP_NAME = sling-sync-wrapper
REGISTRY = registry.local
SLING_CLI_VERSION ?= latest
DUCKDB_CLI_VERSION ?= 0.10.1

all: build

build:
	go build -o bin/$(APP_NAME) ./cmd/wrapper

docker:
	docker build -t $(REGISTRY)/$(APP_NAME):latest .

push: docker
	docker push $(REGISTRY)/$(APP_NAME):latest

run-local:
	MISSION_CLUSTER_ID=local \
	SYNC_JOB_ID=$$(uuidgen) \
	SLING_CONFIG=./pipeline.yaml \
	SLING_STATE=file://./sling_state.json \
	OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
	./bin/$(APP_NAME)


quickstart:
	go run ./cmd/quickstart

install-sling-cli:
	go install github.com/slingdata/sling-cli@$(SLING_CLI_VERSION)

install-duckdb-cli:
	curl -L https://github.com/duckdb/duckdb/releases/download/v$(DUCKDB_CLI_VERSION)/duckdb_cli-linux-amd64.zip -o /tmp/duckdb_cli.zip
	unzip -o /tmp/duckdb_cli.zip -d /usr/local/bin
	chmod +x /usr/local/bin/duckdb
	rm /tmp/duckdb_cli.zip

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

test:
	go test ./...
