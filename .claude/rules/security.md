# Security

Never hardcode absolute paths containing usernames or system-specific directories.
This applies to all files including source code, configuration, and settings files.

```go
// incorrect
path := "/Users/username/personal/spoofdpi/internal"

// correct
home, _ := os.UserHomeDir()
path := filepath.Join(home, ".spoofdpi")
```

```json
// incorrect
{ "command": "cd /Users/username/personal/spoofdpi && go test ./..." }

// correct
{ "command": "go test ./..." }
```
