# AGENTS Instructions

These guidelines apply to the entire repository.

## Development workflow

- Format Go code before committing:
  ```bash
  gofmt -w <files>
  ```
  or run `go fmt ./...` to format everything.
- Keep dependencies tidy with:
  ```bash
  go mod tidy
  ```
- Run static analysis:
  ```bash
  go vet ./...
  ```
- Run unit tests:
  ```bash
  make test
  ```
  This executes `go test ./...` for all packages.
- Build the binary locally with:
  ```bash
  make build
  ```
- Additional useful commands from the Makefile:
  - `make fmt` – format Go code.
  - `make vet` – run static analysis.
  - `make tidy` – tidy module dependencies.
  - `make docker` – build the container image.
  - `make push` – push the image to the registry.
  - `make run-local` – run the wrapper locally using default env vars.
  - `make install-sling-cli` – install the Sling CLI.
  - `make install-duckdb-cli` – install the DuckDB CLI used to inspect quickstart results.
  - `make quickstart` – run the quickstart example and verify the output:
    ```bash
    duckdb quickstart/command.db "select distinct synced_from from telemetry;"
    ```
    The result should list `mission1` and `mission2`.

Ensure tests pass before opening a pull request.

Additional guidelines:

- When making changes, update all relevant documentation, configuration files, Helm charts, and similar assets.
- Write multiple robust tests to verify the behavior.
- Verify the changes by executing the full test suite.
- Use Conventional Commit formatting for commit messages when creating a pull request.
