package floxyctl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rom8726/floxy-pro"
)

type ShellHandler struct {
	name     string
	execPath string
	isScript bool
	debug    bool
}

func NewShellHandler(name, exec string, debug bool) *ShellHandler {
	isScript := false
	if strings.Contains(exec, "\n") {
		if _, err := os.Stat(exec); os.IsNotExist(err) {
			isScript = true
		}
	}

	return &ShellHandler{
		name:     name,
		execPath: exec,
		isScript: isScript,
		debug:    debug,
	}
}

func (h *ShellHandler) Name() string {
	return h.name
}

func (h *ShellHandler) Execute(
	ctx context.Context,
	stepCtx floxy.StepContext,
	input json.RawMessage,
) (json.RawMessage, error) {
	var cmd *exec.Cmd

	if h.isScript {
		scriptContent := "set -e\n" + h.execPath
		cmd = exec.CommandContext(ctx, "bash", "-c", scriptContent)
	} else {
		if _, err := os.Stat(h.execPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("script file not found: %s", h.execPath)
		}
		cmd = exec.CommandContext(ctx, "bash", h.execPath)
	}

	cmd.Env = append(cmd.Env, fmt.Sprintf("INPUT=%s", string(input)))

	var inputData map[string]any
	if err := json.Unmarshal(input, &inputData); err == nil {
		for k, v := range inputData {
			var val string
			if strVal, ok := v.(string); ok {
				val = strVal
			} else {
				val = fmt.Sprintf("%v", v)
			}
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", strings.ToUpper(k), val))
		}
	}

	cmd.Env = append(cmd.Env,
		fmt.Sprintf("FLOXY_INSTANCE_ID=%d", stepCtx.InstanceID()),
		fmt.Sprintf("FLOXY_STEP_NAME=%s", stepCtx.StepName()),
		fmt.Sprintf("FLOXY_IDEMPOTENCY_KEY=%s", stepCtx.IdempotencyKey()),
		fmt.Sprintf("FLOXY_RETRY_COUNT=%d", stepCtx.RetryCount()),
		fmt.Sprintf("FLOXY_DEBUG=%t", h.debug),
	)

	cmd.Env = append(cmd.Env, os.Environ()...)

	if h.debug {
		_, _ = fmt.Fprintf(os.Stderr, "[DEBUG] Handler '%s' input: %s\n", h.name, string(input))
	}

	var stdoutBuf strings.Builder
	var stderrBuf strings.Builder
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderrOutput := stderrBuf.String()
			if h.debug && stderrOutput != "" {
				_, _ = fmt.Fprintf(os.Stderr, "[DEBUG] Handler '%s' stderr: %s\n", h.name, stderrOutput)
			}
			return nil, fmt.Errorf("script execution failed with exit code %d: %w\nStderr: %s", exitErr.ExitCode(), err, stderrOutput)
		}
		stderrOutput := stderrBuf.String()
		stdoutOutput := stdoutBuf.String()
		if stderrOutput != "" || stdoutOutput != "" {
			details := ""
			if stderrOutput != "" {
				details += fmt.Sprintf("\nStderr: %s", stderrOutput)
			}
			if stdoutOutput != "" {
				details += fmt.Sprintf("\nStdout: %s", stdoutOutput)
			}
			return nil, fmt.Errorf("script execution failed: %w%s", err, details)
		}
		return nil, fmt.Errorf("script execution failed: %w", err)
	}

	outputStr := stdoutBuf.String()
	if len(outputStr) == 0 {
		return nil, fmt.Errorf("script produced no output")
	}

	output := []byte(outputStr)

	var result json.RawMessage
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("script output is not valid JSON: %w\nOutput: %s", err, string(output))
	}

	if h.debug {
		_, _ = fmt.Fprintf(os.Stderr, "[DEBUG] Handler '%s' output: %s\n", h.name, string(result))
	}

	return result, nil
}
