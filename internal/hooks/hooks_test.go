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

func silentLogger() *pterm.Logger {
	return pterm.DefaultLogger.WithLevel(pterm.LogLevelDisabled)
}

func TestRunHook_NilHook(t *testing.T) {
	err := RunHook(context.Background(), nil, silentLogger(), map[string]string{})
	assert.NoError(t, err)
}

func TestRunHook_WarnOnFailure_ContinuesOnError(t *testing.T) {
	var run []string
	if runtime.GOOS == "windows" {
		run = []string{"exit /b 1", "echo second"}
	} else {
		run = []string{"exit 1", "echo second"}
	}
	hook := &models.CategoryHook{OnFailure: "warn", Run: run}
	err := RunHook(context.Background(), hook, silentLogger(), map[string]string{})
	assert.NoError(t, err)
}

func TestRunHook_AbortOnFailure_StopsOnError(t *testing.T) {
	tmp := t.TempDir()
	marker := filepath.Join(tmp, "second_ran")

	var run []string
	if runtime.GOOS == "windows" {
		run = []string{
			"exit /b 1",
			fmt.Sprintf("echo x > %s", marker),
		}
	} else {
		run = []string{
			"exit 1",
			fmt.Sprintf("touch %s", marker),
		}
	}
	hook := &models.CategoryHook{OnFailure: "abort", Run: run}
	err := RunHook(context.Background(), hook, silentLogger(), map[string]string{})
	require.Error(t, err)
	assert.NoFileExists(t, marker, "second command should not have run")
}

func TestRunHook_EnvVarsInjected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("env var echo syntax differs on Windows")
	}
	tmp := t.TempDir()
	out := filepath.Join(tmp, "out.txt")
	hook := &models.CategoryHook{
		OnFailure: "abort",
		Run:       []string{fmt.Sprintf(`echo "$ML_CATEGORY" > %s`, out)},
	}
	err := RunHook(context.Background(), hook, silentLogger(), map[string]string{"ML_CATEGORY": "images"})
	require.NoError(t, err)
	data, _ := os.ReadFile(out)
	assert.Contains(t, string(data), "images")
}

func TestDefaultShell_ReturnsNonEmpty(t *testing.T) {
	shell, args := defaultShell("")
	assert.NotEmpty(t, shell)
	assert.NotNil(t, args)
}

func TestDefaultShell_Override(t *testing.T) {
	if runtime.GOOS == "windows" {
		shell, args := defaultShell("pwsh")
		assert.Equal(t, "pwsh", shell)
		assert.Equal(t, []string{"-NonInteractive", "-NoProfile", "-Command"}, args)

		shell, args = defaultShell("cmd")
		assert.Equal(t, "cmd", shell)
		assert.Equal(t, []string{"/C"}, args)
	} else {
		shell, args := defaultShell("bash")
		assert.Equal(t, "bash", shell)
		assert.Equal(t, []string{"-c"}, args)
	}
}

func TestBuildEnv_ContainsInjected(t *testing.T) {
	env := buildEnv(map[string]string{"ML_CATEGORY": "docs", "ML_DRY_RUN": "false"})
	assert.Contains(t, env, "ML_CATEGORY=docs")
	assert.Contains(t, env, "ML_DRY_RUN=false")
}
