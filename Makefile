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
       SLING_TIMEOUT=30m \
        OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
        ./bin/$(APP_NAME)


quickstart: install-sling-cli install-duckdb-cli
	go run ./cmd/quickstart
	duckdb quickstart/command.db "select distinct synced_from from telemetry;"

install-sling-cli:
	@if ! command -v sling >/dev/null 2>&1; then \
	        if [ "$(SLING_CLI_VERSION)" = "latest" ]; then \
	                URL="https://github.com/slingdata-io/sling-cli/releases/latest/download/sling_linux_amd64.tar.gz"; \
	        else \
	                URL="https://github.com/slingdata-io/sling-cli/releases/download/$(SLING_CLI_VERSION)/sling_linux_amd64.tar.gz"; \
	        fi; \
	        curl -L $$URL -o /tmp/sling_cli.tar.gz; \
	        tar -C /usr/local/bin -xzf /tmp/sling_cli.tar.gz sling; \
	        chmod +x /usr/local/bin/sling; \
	        rm /tmp/sling_cli.tar.gz; \
	else \
	        echo "sling already installed"; \
	fi

install-duckdb-cli:
	@if ! command -v duckdb >/dev/null 2>&1; then \
	        curl -L https://github.com/duckdb/duckdb/releases/download/v$(DUCKDB_CLI_VERSION)/duckdb_cli-linux-amd64.zip -o /tmp/duckdb_cli.zip; \
	        unzip -o /tmp/duckdb_cli.zip -d /usr/local/bin; \
	        chmod +x /usr/local/bin/duckdb; \
	        rm /tmp/duckdb_cli.zip; \
	else \
	        echo "duckdb already installed"; \
	fi

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

test:
	go test ./...
