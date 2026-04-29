package executil

import (
	"fmt"
	"os/exec"

	"mvdan.cc/sh/shell"
)

func Commandf(format string, args ...any) (string, error) {
	fullCmd := fmt.Sprintf(format, args...)

	splitArgs, err := shell.Fields(fullCmd, nil)
	if err != nil {
		return "", err
	}

	if len(splitArgs) == 0 {
		return "", nil
	}

	out, err := exec.Command(splitArgs[0], splitArgs[1:]...).CombinedOutput()
	return string(out), err
}
