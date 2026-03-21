package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
)

var versionRegex = regexp.MustCompile(` \| !V([0-9]+\.[0-9]+\.[0-9]+);`)

func main() {
	args := os.Args[1:]
	data, err := readInput()
	if err != nil {
		printError(err)
		printUsage()
		os.Exit(1)
	}

	version, err := detectVersion(data)
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	binPath, err := ensureSplitans(version)
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	if err := runSplitans(binPath, args, data); err != nil {
		if exitErr := newExitCodeError(err); exitErr != nil {
			os.Exit(exitErr.code)
		}
		printError(err)
		os.Exit(1)
	}
}

func readInput() ([]byte, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return nil, errors.New("stdin required")
	}
	return io.ReadAll(os.Stdin)
}

func detectVersion(data []byte) (string, error) {
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		cleanLine := strings.TrimRight(line, "\r")
		trimmed := strings.TrimLeft(cleanLine, " \t")
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		matches := versionRegex.FindStringSubmatch(cleanLine)
		if len(matches) != 2 {
			return "", errors.New("missing neotex !V metadata")
		}
		return matches[1], nil
	}
	return "", errors.New("missing neotex !V metadata")
}

func ensureSplitans(version string) (string, error) {
	cacheDir, binPath, err := cachePaths(version)
	if err != nil {
		return "", err
	}
	if isExecutable(binPath) {
		return binPath, nil
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}
	cmd := exec.Command("go", "install", "github.com/badele/splitans/cmd/splitans@v"+version)
	cmd.Env = append(os.Environ(), "GOBIN="+cacheDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	if !isExecutable(binPath) {
		return "", fmt.Errorf("splitans binary not found after install: %s", binPath)
	}
	return binPath, nil
}

func cachePaths(version string) (string, string, error) {
	root, err := os.UserCacheDir()
	if err != nil || root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
		root = filepath.Join(home, ".cache")
	}
	cacheDir := filepath.Join(root, "splitans", version)
	return cacheDir, filepath.Join(cacheDir, binaryName()), nil
}

func binaryName() string {
	name := "splitans"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0o111 != 0
}

func runSplitans(binPath string, args []string, data []byte) error {
	cmd := exec.Command(binPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = bytes.NewReader(data)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return &exitCodeError{code: status.ExitStatus()}
			}
		}
		return err
	}
	return nil
}

type exitCodeError struct {
	code int
}

func (err *exitCodeError) Error() string {
	return fmt.Sprintf("splitans exited with status %d", err.code)
}

func newExitCodeError(err error) *exitCodeError {
	var exitErr *exitCodeError
	if errors.As(err, &exitErr) {
		return exitErr
	}
	return nil
}

func printError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: splitansx [splitans args] < input.neo")
}
