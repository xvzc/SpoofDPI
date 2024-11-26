package util

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

func loadPatternsFromFile(path string) ([]*regexp.Regexp, error) {
	if path == "" {
		return nil, nil
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("pattern file path: %w", err)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening pattern file: %w", err)
	}
	defer func() {
		if e := file.Close(); e != nil && err == nil {
			err = e
		}
	}()

	var patterns []*regexp.Regexp
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		pattern := regexp.MustCompile(scanner.Text())
		patterns = append(patterns, pattern)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parsing pattern file: %w", err)
	}

	return patterns, nil
}