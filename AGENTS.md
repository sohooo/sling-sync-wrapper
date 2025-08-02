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
  - `make docker` – build the container image.
  - `make push` – push the image to the registry.
  - `make run-local` – run the wrapper locally using default env vars.
  - `make quickstart` – run the quickstart example.

Ensure tests pass before opening a pull request.
