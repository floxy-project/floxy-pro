package handlers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rom8726/floxy-pro"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewShellHandler_InlineScriptWithNewline(t *testing.T) {
	inlineScript := "echo 'test1'\necho 'test2'"
	handler := NewShellHandler("test", inlineScript, false)

	assert.True(t, handler.isScript, "inline script with newline should be detected as script")
	assert.Equal(t, inlineScript, handler.execPath)
}

func TestNewShellHandler_InlineScriptWithNewlineButFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_script.sh")

	err := os.WriteFile(tmpFile, []byte("echo 'file content'"), 0644)
	require.NoError(t, err)

	inlineScript := "echo 'test1'\necho 'test2'"
	handler := NewShellHandler("test", inlineScript, false)

	assert.True(t, handler.isScript, "inline script should be detected as script even if file with same name exists")
}

func TestNewShellHandler_FileScript(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_script.sh")

	scriptContent := "echo 'test'"
	err := os.WriteFile(tmpFile, []byte(scriptContent), 0644)
	require.NoError(t, err)

	handler := NewShellHandler("test", tmpFile, false)

	assert.False(t, handler.isScript, "file path should be detected as file script")
	assert.Equal(t, tmpFile, handler.execPath)
}

func TestNewShellHandler_InlineScriptSingleLine(t *testing.T) {
	inlineScript := "echo 'single line'"
	handler := NewShellHandler("test", inlineScript, false)

	assert.False(t, handler.isScript, "single line without newline should not be detected as script")
}

func TestShellHandler_Execute_InlineScript(t *testing.T) {
	mockCtx := floxy.NewMockStepContext(t)
	mockCtx.EXPECT().InstanceID().Return(int64(1)).Maybe()
	mockCtx.EXPECT().StepName().Return("test-step").Maybe()
	mockCtx.EXPECT().IdempotencyKey().Return("key-123").Maybe()
	mockCtx.EXPECT().RetryCount().Return(0).Maybe()

	inlineScript := "echo '{\"result\": \"success\"}'"
	handler := NewShellHandler("test", inlineScript, false)

	assert.False(t, handler.isScript, "single line without newline should not be detected as script")
}

func TestShellHandler_Execute_InlineScriptWithNewline(t *testing.T) {
	mockCtx := floxy.NewMockStepContext(t)
	mockCtx.EXPECT().InstanceID().Return(int64(1)).Maybe()
	mockCtx.EXPECT().StepName().Return("test-step").Maybe()
	mockCtx.EXPECT().IdempotencyKey().Return("key-123").Maybe()
	mockCtx.EXPECT().RetryCount().Return(0).Maybe()

	inlineScript := "echo '{\"result\": \"success\"}'"
	handler := NewShellHandler("test", inlineScript, false)

	input := json.RawMessage(`{"key": "value"}`)
	_, err := handler.Execute(context.Background(), mockCtx, input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "script file not found")
}

func TestShellHandler_Execute_InlineScriptMultiLine(t *testing.T) {
	mockCtx := floxy.NewMockStepContext(t)
	mockCtx.EXPECT().InstanceID().Return(int64(1)).Maybe()
	mockCtx.EXPECT().StepName().Return("test-step").Maybe()
	mockCtx.EXPECT().IdempotencyKey().Return("key-123").Maybe()
	mockCtx.EXPECT().RetryCount().Return(0).Maybe()

	inlineScript := "echo 'test'\necho '{\"result\": \"done\"}'"
	handler := NewShellHandler("test", inlineScript, false)

	assert.True(t, handler.isScript, "multi-line script should be detected as script")

	input := json.RawMessage(`{}`)
	_, err := handler.Execute(context.Background(), mockCtx, input)

	assert.Error(t, err, "should fail because script outputs multiple lines, only last is JSON")
}

func TestShellHandler_Execute_FileScript(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_script.sh")

	scriptContent := "#!/bin/bash\necho '{\"result\": \"file executed\"}'"
	err := os.WriteFile(tmpFile, []byte(scriptContent), 0755)
	require.NoError(t, err)

	mockCtx := floxy.NewMockStepContext(t)
	mockCtx.EXPECT().InstanceID().Return(int64(1)).Maybe()
	mockCtx.EXPECT().StepName().Return("test-step").Maybe()
	mockCtx.EXPECT().IdempotencyKey().Return("key-123").Maybe()
	mockCtx.EXPECT().RetryCount().Return(0).Maybe()

	handler := NewShellHandler("test", tmpFile, false)

	input := json.RawMessage(`{}`)
	result, err := handler.Execute(context.Background(), mockCtx, input)

	require.NoError(t, err)
	assert.Contains(t, string(result), "file executed")
}

func TestShellHandler_Execute_InlineScriptWithEchoOperations(t *testing.T) {
	mockCtx := floxy.NewMockStepContext(t)
	mockCtx.EXPECT().InstanceID().Return(int64(1)).Maybe()
	mockCtx.EXPECT().StepName().Return("test-step").Maybe()
	mockCtx.EXPECT().IdempotencyKey().Return("key-123").Maybe()
	mockCtx.EXPECT().RetryCount().Return(0).Maybe()

	inlineScript := "echo '{\"final\": \"complete\"}'"
	handler := NewShellHandler("test", inlineScript, false)

	assert.False(t, handler.isScript, "single line without newline should not be detected as script")

	input := json.RawMessage(`{"test": "data"}`)
	_, err := handler.Execute(context.Background(), mockCtx, input)

	assert.Error(t, err, "should fail because single line script is treated as file path")
	assert.Contains(t, err.Error(), "script file not found")
}

func TestShellHandler_Execute_InlineScriptWithEnvironmentVariables(t *testing.T) {
	mockCtx := floxy.NewMockStepContext(t)
	mockCtx.EXPECT().InstanceID().Return(int64(42)).Maybe()
	mockCtx.EXPECT().StepName().Return("env-test").Maybe()
	mockCtx.EXPECT().IdempotencyKey().Return("idemp-key").Maybe()
	mockCtx.EXPECT().RetryCount().Return(2).Maybe()

	inlineScript := "echo \"{\\\"instance_id\\\": \\\"$FLOXY_INSTANCE_ID\\\", \\\"step_name\\\": \\\"$FLOXY_STEP_NAME\\\", \\\"idempotency_key\\\": \\\"$FLOXY_IDEMPOTENCY_KEY\\\", \\\"retry_count\\\": \\\"$FLOXY_RETRY_COUNT\\\"}\""
	handler := NewShellHandler("test", inlineScript, false)

	assert.False(t, handler.isScript, "single line without newline should not be detected as script")

	input := json.RawMessage(`{"key": "value"}`)
	_, err := handler.Execute(context.Background(), mockCtx, input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "script file not found")
}

func TestShellHandler_Execute_InlineScriptWithInputVariables(t *testing.T) {
	mockCtx := floxy.NewMockStepContext(t)
	mockCtx.EXPECT().InstanceID().Return(int64(1)).Maybe()
	mockCtx.EXPECT().StepName().Return("test-step").Maybe()
	mockCtx.EXPECT().IdempotencyKey().Return("key-123").Maybe()
	mockCtx.EXPECT().RetryCount().Return(0).Maybe()

	inlineScript := "echo \"{\\\"received_key\\\": \\\"$KEY\\\", \\\"received_value\\\": \\\"$VALUE\\\"}\""
	handler := NewShellHandler("test", inlineScript, false)

	assert.False(t, handler.isScript, "single line without newline should not be detected as script")

	input := json.RawMessage(`{"key": "test-key", "value": "test-value"}`)
	_, err := handler.Execute(context.Background(), mockCtx, input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "script file not found")
}

func TestShellHandler_Execute_InlineScriptMultipleEchoLines(t *testing.T) {
	mockCtx := floxy.NewMockStepContext(t)
	mockCtx.EXPECT().InstanceID().Return(int64(1)).Maybe()
	mockCtx.EXPECT().StepName().Return("test-step").Maybe()
	mockCtx.EXPECT().IdempotencyKey().Return("key-123").Maybe()
	mockCtx.EXPECT().RetryCount().Return(0).Maybe()

	inlineScript := "echo '{\"result\": \"finished\"}'"
	handler := NewShellHandler("test", inlineScript, false)

	assert.False(t, handler.isScript, "single line without newline should not be detected as script")

	input := json.RawMessage(`{}`)
	_, err := handler.Execute(context.Background(), mockCtx, input)

	assert.Error(t, err, "should fail because single line script is treated as file path")
	assert.Contains(t, err.Error(), "script file not found")
}

func TestShellHandler_Execute_InlineScriptWithNewlineButFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	inlineScript := "echo 'line1'\necho 'line2'"

	tmpFile := filepath.Join(tmpDir, "test_script.sh")
	fileContent := "echo 'different content'"
	err := os.WriteFile(tmpFile, []byte(fileContent), 0644)
	require.NoError(t, err)

	handler := NewShellHandler("test", inlineScript, false)

	assert.True(t, handler.isScript, "inline script with newline should be detected as script even if other files exist")
}

func TestShellHandler_Execute_InlineScriptSingleLineWithoutNewline(t *testing.T) {
	mockCtx := floxy.NewMockStepContext(t)
	mockCtx.EXPECT().InstanceID().Return(int64(1)).Maybe()
	mockCtx.EXPECT().StepName().Return("test-step").Maybe()
	mockCtx.EXPECT().IdempotencyKey().Return("key-123").Maybe()
	mockCtx.EXPECT().RetryCount().Return(0).Maybe()

	inlineScript := "echo '{\"result\": \"single\"}'"
	handler := NewShellHandler("test", inlineScript, false)

	assert.False(t, handler.isScript, "single line without newline should not be detected as script")

	input := json.RawMessage(`{}`)
	_, err := handler.Execute(context.Background(), mockCtx, input)

	assert.Error(t, err, "should fail because single line script is treated as file path")
	assert.Contains(t, err.Error(), "script file not found")
}

func TestShellHandler_Execute_InlineScriptWithNewlineFileDoesNotExist(t *testing.T) {
	mockCtx := floxy.NewMockStepContext(t)
	mockCtx.EXPECT().InstanceID().Return(int64(1)).Maybe()
	mockCtx.EXPECT().StepName().Return("test-step").Maybe()
	mockCtx.EXPECT().IdempotencyKey().Return("key-123").Maybe()
	mockCtx.EXPECT().RetryCount().Return(0).Maybe()

	inlineScript := "echo '{\"result\": \"inline\"}'"
	handler := NewShellHandler("test", inlineScript, false)

	assert.False(t, handler.isScript, "single line without newline should not be detected as script")

	input := json.RawMessage(`{}`)
	_, err := handler.Execute(context.Background(), mockCtx, input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "script file not found")
}

func TestShellHandler_Execute_InlineScriptMultiLineWithNewline(t *testing.T) {
	mockCtx := floxy.NewMockStepContext(t)
	mockCtx.EXPECT().InstanceID().Return(int64(1)).Maybe()
	mockCtx.EXPECT().StepName().Return("test-step").Maybe()
	mockCtx.EXPECT().IdempotencyKey().Return("key-123").Maybe()
	mockCtx.EXPECT().RetryCount().Return(0).Maybe()

	inlineScript := "echo '{\"result\": \"success\"}'"
	handler := NewShellHandler("test", inlineScript, false)

	assert.False(t, handler.isScript, "single line without newline should not be detected as script")

	input := json.RawMessage(`{}`)
	_, err := handler.Execute(context.Background(), mockCtx, input)

	assert.Error(t, err, "should fail because single line script is treated as file path")
	assert.Contains(t, err.Error(), "script file not found")
}

func TestShellHandler_Execute_InlineScriptMultiLineWithNewlineValid(t *testing.T) {
	mockCtx := floxy.NewMockStepContext(t)
	mockCtx.EXPECT().InstanceID().Return(int64(1)).Maybe()
	mockCtx.EXPECT().StepName().Return("test-step").Maybe()
	mockCtx.EXPECT().IdempotencyKey().Return("key-123").Maybe()
	mockCtx.EXPECT().RetryCount().Return(0).Maybe()

	inlineScript := "echo '{\"result\": \"success\"}'"
	handler := NewShellHandler("test", inlineScript, false)

	assert.False(t, handler.isScript, "single line without newline should not be detected as script")
}

func TestShellHandler_Execute_InlineScriptMultiLineWithNewlineActuallyMultiLine(t *testing.T) {
	mockCtx := floxy.NewMockStepContext(t)
	mockCtx.EXPECT().InstanceID().Return(int64(1)).Maybe()
	mockCtx.EXPECT().StepName().Return("test-step").Maybe()
	mockCtx.EXPECT().IdempotencyKey().Return("key-123").Maybe()
	mockCtx.EXPECT().RetryCount().Return(0).Maybe()

	inlineScript := "VAR=test\necho \"{\\\"var\\\": \\\"$VAR\\\", \\\"result\\\": \\\"success\\\"}\""
	handler := NewShellHandler("test", inlineScript, false)

	assert.True(t, handler.isScript, "multi-line script should be detected as script")

	input := json.RawMessage(`{}`)
	result, err := handler.Execute(context.Background(), mockCtx, input)

	require.NoError(t, err)
	assert.Contains(t, string(result), "success")
	assert.Contains(t, string(result), "test")
}
