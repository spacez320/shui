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
	CRYPTARCH_BINARY_DIR  = "dist"      // Directory the executable is in.
	CRYPTARCH_BINARY_NAME = "cryptarch" // Name of the Cryptarch binary.
)

var (
	// Filepath for the Cryptarch executable.
	cryptarch = filepath.Join(CRYPTARCH_BINARY_DIR, CRYPTARCH_BINARY_NAME)
	// Environment to execute Cryptarch with, appended to `os.Environ()`.
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

// Builds the Cryptarch binary.
func buildCryptarch() ([]byte, error) {
	cmd := exec.Command("go", "build", "-o", cryptarch)
	return cmd.CombinedOutput()
}

// Executes the Cryptarch binary.
func runCryptarch(args []string) ([]byte, error) {
	var (
		environ = append(os.Environ(), environToA(testEnviron))
	)

	cmd := exec.Command(cryptarch, args...)
	cmd.Env = environ
	return cmd.CombinedOutput()
}

// Test set-up.
func TestMain(m *testing.M) {
	buildCryptarch()
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
			_, err := runCryptarch(cliTest.cliArgs)

			if err != nil {
				t.Error(err)
			}
		})
	}
}
