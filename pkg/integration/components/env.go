package components

import (
	"fmt"
	"os"
)

const (
	// These values will be passed to lazygit
	LAZYGIT_ROOT_DIR          = "LAZYGIT_ROOT_DIR"
	SANDBOX_ENV_VAR           = "SANDBOX"
	TEST_NAME_ENV_VAR         = "TEST_NAME"
	WAIT_FOR_DEBUGGER_ENV_VAR = "WAIT_FOR_DEBUGGER"

	// These values will be passed to both lazygit and shell commands
	GIT_CONFIG_GLOBAL_ENV_VAR = "GIT_CONFIG_GLOBAL"
	// We pass PWD because if it's defined, Go will use it as the working directory
	// rather than make a syscall to the OS, and that means symlinks won't be resolved,
	// which is good to test for.
	PWD = "PWD"

	// These values will be passed through to lazygit and shell commands, with their
	// values inherited from the host environment
	PATH = "PATH"
	TERM = "TERM"
)

// Tests will inherit these environment variables from the host environment, rather
// than the test runner deciding the values itself.
// All other environment variables present in the host environment will be ignored.
// Having such a minimal list ensures that lazygit behaves the same across different test environments.
var hostEnvironmentAllowlist = [...]string{
	PATH,
	TERM,
}

// Returns a copy of the environment filtered by
// hostEnvironmentAllowlist
func AllowedHostEnvironment() []string {
	env := []string{}
	for _, envVar := range hostEnvironmentAllowlist {
		env = append(env, fmt.Sprintf("%s=%s", envVar, os.Getenv(envVar)))
	}
	return env
}
