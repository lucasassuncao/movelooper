package hooks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lucasassuncao/movelooper/internal/models"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testRunHook defines the structure for test cases of the RunHook function,
// containing a hook builder (receives a temp dir), env vars, an optional skip condition,
// an error expectation flag, and a check function for additional assertions.
type testRunHook struct {
	name    string
	hook    func(dir string) *models.CategoryHook
	env     map[string]string
	skip    func() bool
	wantErr bool
	check   func(t *testing.T, dir string)
}

// testRunHookTestCases defines a set of test cases for the RunHook function,
// covering nil hook, warn-on-failure, abort-on-failure, and env var injection.
var testRunHookTestCases = []testRunHook{
	{
		name: "nil hook returns no error",
		hook: func(dir string) *models.CategoryHook { return nil },
	},
	{
		name: "warn on failure continues on error",
		hook: func(dir string) *models.CategoryHook {
			var run []string
			if runtime.GOOS == "windows" {
				run = []string{"exit /b 1", "echo second"}
			} else {
				run = []string{"exit 1", "echo second"}
			}
			return &models.CategoryHook{
				OnFailure: "warn",
				Run:       run,
			}
		},
	},
	{
		name: "abort on failure stops on error",
		hook: func(dir string) *models.CategoryHook {
			marker := filepath.Join(dir, "second_ran")
			var run []string
			if runtime.GOOS == "windows" {
				run = []string{"exit /b 1", fmt.Sprintf("echo x > %s", marker)}
			} else {
				run = []string{"exit 1", fmt.Sprintf("touch %s", marker)}
			}
			return &models.CategoryHook{
				OnFailure: "abort",
				Run:       run,
			}
		},
		wantErr: true,
		check: func(t *testing.T, dir string) {
			assert.NoFileExists(t, filepath.Join(dir, "second_ran"), "second command should not have run")
		},
	},
	{
		name: "env vars are injected into hook commands",
		skip: func() bool { return runtime.GOOS == "windows" },
		hook: func(dir string) *models.CategoryHook {
			out := filepath.Join(dir, "out.txt")
			return &models.CategoryHook{
				OnFailure: "abort",
				Run:       []string{fmt.Sprintf(`echo "$ML_CATEGORY" > %s`, out)},
			}
		},
		env: map[string]string{"ML_CATEGORY": "images"},
		check: func(t *testing.T, dir string) {
			data, _ := os.ReadFile(filepath.Join(dir, "out.txt"))
			assert.Contains(t, string(data), "images")
		},
	},
}

// TestRunHook tests the RunHook function with various hook configurations
// to ensure it correctly executes commands and handles failures.
func TestRunHook(t *testing.T) {
	for _, tt := range testRunHookTestCases {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip != nil && tt.skip() {
				t.Skip()
			}
			dir := t.TempDir()
			hook := tt.hook(dir)
			env := tt.env
			if env == nil {
				env = map[string]string{}
			}

			err := RunHook(context.Background(), hook, silentLogger(), env)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.check != nil {
				tt.check(t, dir)
			}
		})
	}
}

// testDefaultShell defines the structure for test cases of the defaultShell function,
// containing the override value, expected shell, expected args, and an optional skip condition.
type testDefaultShell struct {
	name      string
	override  string
	skip      func() bool
	wantShell string
	wantArgs  []string
}

// testDefaultShellTestCases defines a set of test cases for the defaultShell function,
// covering the default (no override) and platform-specific override scenarios.
var testDefaultShellTestCases = []testDefaultShell{
	{
		name:      "no override returns non-empty shell",
		override:  "",
		wantShell: "",
		wantArgs:  nil,
	},
	{
		name:      "pwsh override on windows",
		override:  "pwsh",
		skip:      func() bool { return runtime.GOOS != "windows" },
		wantShell: "pwsh",
		wantArgs:  []string{"-NonInteractive", "-NoProfile", "-Command"},
	},
	{
		name:      "cmd override on windows",
		override:  "cmd",
		skip:      func() bool { return runtime.GOOS != "windows" },
		wantShell: "cmd",
		wantArgs:  []string{"/C"},
	},
	{
		name:      "bash override on unix",
		override:  "bash",
		skip:      func() bool { return runtime.GOOS == "windows" },
		wantShell: "bash",
		wantArgs:  []string{"-c"},
	},
}

// TestDefaultShell tests the defaultShell function with various override values
// to ensure it returns the correct shell and arguments for each platform.
func TestDefaultShell(t *testing.T) {
	for _, tt := range testDefaultShellTestCases {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip != nil && tt.skip() {
				t.Skip()
			}
			shell, args := defaultShell(tt.override)
			assert.NotEmpty(t, shell)
			assert.NotNil(t, args)
			if tt.wantShell != "" {
				assert.Equal(t, tt.wantShell, shell)
				assert.Equal(t, tt.wantArgs, args)
			}
		})
	}
}

// testBuildEnv defines the structure for test cases of the buildEnv function,
// containing the extra env vars to inject and the expected strings in the result.
type testBuildEnv struct {
	name   string
	env    map[string]string
	wantIn []string
}

// testBuildEnvTestCases defines a set of test cases for the buildEnv function,
// covering single and multiple injected env vars.
var testBuildEnvTestCases = []testBuildEnv{
	{
		name:   "injected vars appear in result",
		env:    map[string]string{"ML_CATEGORY": "docs", "ML_DRY_RUN": "false"},
		wantIn: []string{"ML_CATEGORY=docs", "ML_DRY_RUN=false"},
	},
}

// TestBuildEnv tests the buildEnv function to ensure it correctly merges
// the process environment with the provided extra vars.
func TestBuildEnv(t *testing.T) {
	for _, tt := range testBuildEnvTestCases {
		t.Run(tt.name, func(t *testing.T) {
			env := buildEnv(tt.env)
			for _, want := range tt.wantIn {
				assert.Contains(t, env, want)
			}
		})
	}
}

func silentLogger() *pterm.Logger {
	return pterm.DefaultLogger.WithLevel(pterm.LogLevelDisabled)
}
