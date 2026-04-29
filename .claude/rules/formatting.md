# Formatting

Code must be formatted with `gofmt`. After any code change:
- Check: `gofmt -l .` from the project root (no output means OK)
- Format: `gofmt -w .` from the project root

Run `golangci-lint run` from the project root for additional linting; configuration lives in `.golangci.yml`.

All formatting and lint checks must pass before considering the task complete.
