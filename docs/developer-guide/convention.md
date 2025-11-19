# Convention

## Commit Messages

This project adheres to the [Conventional Commits](https://www.conventionalcommits.org) specification to ensure a clear and standardized project history. Please follow the format below.

```text
type(scope): description

[optional body]

[optional footer(s)]
```

!!! note "Allowed valus for `<type>`"
    - **feat:** A new feature
    - **fix:** A bug fix
    - **docs:**	Documentation only changes
    - **style:** Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)
    - **refactor:** A code change that neither fixes a bug nor adds a feature
    - **perf:** A code change that improves performance
    - **test:**	Adding missing tests or correcting existing tests
    - **chore:** Other changes that don't modify src or test files
    - **ci:** Changes to our CI configuration files and scripts
    - **revert:** Reverts a previous commit

## Code Quality

This project utilizes [golangci-lint](https://github.com/golangci/golangci-lint) as the **single entry point** for both code quality checks (linting) and style enforcement (formatting). This ensures consistent code standards across the entire project.

```console
- Format the code (using internal tools like goimports, gofmt)
$ golangci-lint fmt

- Run all quality checks
$ golangci-lint run
```
