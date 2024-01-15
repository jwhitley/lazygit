package components

import (
	"fmt"
	"os"
)

const (
	GIT_CONFIG_GLOBAL_ENV_VAR = "GIT_CONFIG_GLOBAL"
	LAZYGIT_ROOT_DIR          = "LAZYGIT_ROOT_DIR"
	PATH                      = "PATH"
	PWD                       = "PWD"
	SANDBOX_ENV_VAR           = "SANDBOX"
	TERM                      = "TERM"
	TEST_NAME_ENV_VAR         = "TEST_NAME"
	WAIT_FOR_DEBUGGER_ENV_VAR = "WAIT_FOR_DEBUGGER"
)

// These environment variables must be set for lazygit to run.
// As a result, each variable in this list will be passed through
// to integration tests.
var environmentAllowlist = [...]string{
	GIT_CONFIG_GLOBAL_ENV_VAR,
	PATH,
	TERM,
}

// Returns a copy of the environment filtered by
// environmentWhitelist
func TestEnvironment() []string {
	env := []string{}
	for _, envVar := range environmentAllowlist {
		env = append(env, fmt.Sprintf("%s=%s", envVar, os.Getenv(envVar)))
	}
	return env
}
