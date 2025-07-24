APP_NAME = sling-sync-wrapper
REGISTRY = registry.local

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

