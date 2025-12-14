# Code Quality

This project utilizes [golangci-lint](https://github.com/golangci/golangci-lint) as the **single entry point** for both code quality checks (linting) and style enforcement (formatting). This ensures consistent code standards across the entire project.

```console
- Format the code (using internal tools like goimports, gofmt)
$ golangci-lint fmt

- Run all quality checks
$ golangci-lint run
```
