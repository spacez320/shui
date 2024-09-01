package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	SHUI_BINARY_DIR  = "dist" // Directory the executable is in.
	SHUI_BINARY_NAME = "shui" // Name of the Shui binary.
)

var (
	// Filepath for the Shui executable.
	shuiExec = filepath.Join(SHUI_BINARY_DIR, SHUI_BINARY_NAME)
	// Environment to execute Shui with, appended to `os.Environ()`.
	testEnviron = map[string]string{
		"GOCOVERDIR": ".coverdata",
	}
)

// Converts a string-to-string mapping to a comma-delimited string.
func environToA(environ map[string]string) (a string) {
	for k, v := range environ {
		a += fmt.Sprintf("%s=\"%s\"%s", k, v, ",")
	}

	// Take the last delimiter off.
	return strings.TrimSuffix(a, ",")
}

// Builds the Shui binary.
func buildShui() ([]byte, error) {
	cmd := exec.Command("go", "build", "-o", shuiExec)
	return cmd.CombinedOutput()
}

// Executes the Shui binary.
func runShui(args []string) ([]byte, error) {
	var (
		environ = append(os.Environ(), environToA(testEnviron))
	)

	cmd := exec.Command(shuiExec, args...)
	cmd.Env = environ
	return cmd.CombinedOutput()
}

// Test set-up.
func TestMain(m *testing.M) {
	buildShui()
	os.Exit(m.Run())
}

// Run CLI invocation tests.
func TestCLI(t *testing.T) {
	cliTests := []struct {
		testName string
		cliArgs  []string
	}{
		{"help", []string{"-h"}},
	}

	for _, cliTest := range cliTests {
		t.Run(cliTest.testName, func(t *testing.T) {
			_, err := runShui(cliTest.cliArgs)

			if err != nil {
				t.Error(err)
			}
		})
	}
}
