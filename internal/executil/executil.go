package executil

import (
	"fmt"
	"os/exec"
	"strings"

	"mvdan.cc/sh/shell"
)

// Quote wraps s in single quotes so shell.Fields treats it as a single literal
// argument, escaping any embedded single quotes.
func Quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func Commandf(format string, args ...any) (string, error) {
	return Command(fmt.Sprintf(format, args...))
}

func Command(cmd string) (string, error) {
	args, err := shell.Fields(cmd, nil)
	if err != nil {
		return "", err
	}
	if len(args) == 0 {
		return "", nil
	}
	out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	return string(out), err
}
