# Testing

This document outlines how to run tests and the conventions for writing new tests in SpoofDPI.

## Running Tests

To run all tests in the project, use the standard `go test` command from the root directory:

```console
$ go test ./...
```

To run tests with verbose output (showing all test names):

```console
$ go test -v ./...
```

To run a specific test function (e.g., `TestCreateCommand_Flags`):

```console
$ go test -v -run TestCreateCommand_Flags ./internal/config/
```

## Conventions

SpoofDPI uses the standard `testing` package enhanced by [testify](https://github.com/stretchr/testify) for assertions and requirements.

### 1. Framework

- Use `testify/assert` for general assertions where a failure should record an error but continue the test.
- Use `testify/require` for checks that must pass for the rest of the test to proceed (e.g., checking for errors after a function call, ensuring setup was successful).

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {
    result, err := SomeFunction()
    
    // Stop immediately if there's an error
    require.NoError(t, err)
    
    // Assert the result, but continue even if it fails
    assert.Equal(t, "expected", result)
}
```

### 2. Table-Driven Tests

For testing multiple scenarios of the same logic, **table-driven tests** are strongly preferred. This keeps the test code clean and makes it easy to add new cases.

```go
func TestCalculation(t *testing.T) {
    tcs := []struct {
        name     string
        input    int
        expected int
    }{
        {
            name:     "positive number",
            input:    5,
            expected: 10,
        },
        {
            name:     "zero",
            input:    0,
            expected: 0,
        },
    }

    for _, tc := range tcs {
        t.Run(tc.name, func(t *testing.T) {
            result := calculate(tc.input)
            assert.Equal(t, tc.expected, result)
        })
    }
}
```

### 3. Naming

- Test functions should start with `Test` followed by the name of the function or feature being tested (e.g., `TestCreateCommand`).
- Use clear and descriptive names for table-driven test cases (e.g., `name: "default values (no flags)"`).
