package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLintCommand(t *testing.T) {
	// Create temp directory for test files
	tmpDir := t.TempDir()

	t.Run("Valid EventSource", func(t *testing.T) {
		validEventSource := `apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: test-webhook
spec:
  webhook:
    example:
      port: "12000"
      endpoint: /example
      method: POST
`
		testFile := filepath.Join(tmpDir, "valid-eventsource.yaml")
		err := os.WriteFile(testFile, []byte(validEventSource), 0644)
		require.NoError(t, err)

		// Test linting valid file
		hasErrors := false
		hasWarnings := false
		err = lintFile(testFile, &hasErrors, &hasWarnings)
		assert.NoError(t, err)
		assert.False(t, hasErrors)
	})

	t.Run("Invalid EventSource", func(t *testing.T) {
		invalidEventSource := `apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: test-webhook
spec:
  webhook:
    example:
      port: "12000"
      # Missing required endpoint field
      method: POST
`
		testFile := filepath.Join(tmpDir, "invalid-eventsource.yaml")
		err := os.WriteFile(testFile, []byte(invalidEventSource), 0644)
		require.NoError(t, err)

		// Test linting invalid file
		hasErrors := false
		hasWarnings := false
		err = lintFile(testFile, &hasErrors, &hasWarnings)
		// Should not return error, but hasErrors should be true
		assert.NoError(t, err)
		assert.True(t, hasErrors)
	})

	t.Run("Valid Sensor", func(t *testing.T) {
		validSensor := `apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: test-sensor
spec:
  dependencies:
    - name: test-dep
      eventSourceName: test-webhook
      eventName: example
  triggers:
    - template:
        name: trigger-1
        k8s:
          operation: create
          source:
            resource:
              apiVersion: v1
              kind: ConfigMap
              metadata:
                name: test-cm
              data:
                key: value
`
		testFile := filepath.Join(tmpDir, "valid-sensor.yaml")
		err := os.WriteFile(testFile, []byte(validSensor), 0644)
		require.NoError(t, err)

		// Test linting valid file
		hasErrors := false
		hasWarnings := false
		err = lintFile(testFile, &hasErrors, &hasWarnings)
		assert.NoError(t, err)
		assert.False(t, hasErrors)
	})

	t.Run("Multi-Document YAML", func(t *testing.T) {
		multiDoc := `apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: test-webhook
spec:
  webhook:
    example:
      port: "12000"
      endpoint: /example
      method: POST
---
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: test-sensor
spec:
  dependencies:
    - name: test-dep
      eventSourceName: test-webhook
      eventName: example
  triggers:
    - template:
        name: trigger-1
        k8s:
          operation: create
          source:
            resource:
              apiVersion: v1
              kind: ConfigMap
              metadata:
                name: test-cm
              data:
                key: value
`
		testFile := filepath.Join(tmpDir, "multi-doc.yaml")
		err := os.WriteFile(testFile, []byte(multiDoc), 0644)
		require.NoError(t, err)

		// Test linting multi-document file
		hasErrors := false
		hasWarnings := false
		err = lintFile(testFile, &hasErrors, &hasWarnings)
		assert.NoError(t, err)
		assert.False(t, hasErrors)
	})

	t.Run("Non-Argo-Events Resource", func(t *testing.T) {
		otherResource := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
data:
  key: value
`
		testFile := filepath.Join(tmpDir, "configmap.yaml")
		err := os.WriteFile(testFile, []byte(otherResource), 0644)
		require.NoError(t, err)

		// Test linting non-argo-events resource (should be ignored)
		hasErrors := false
		hasWarnings := false
		err = lintFile(testFile, &hasErrors, &hasWarnings)
		assert.NoError(t, err)
		assert.False(t, hasErrors)
	})

	t.Run("Directory Linting", func(t *testing.T) {
		// Create subdirectory
		subDir := filepath.Join(tmpDir, "subdir")
		err := os.MkdirAll(subDir, 0755)
		require.NoError(t, err)

		// Create valid file in subdirectory
		validEventSource := `apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: test-calendar
spec:
  calendar:
    example:
      interval: 10s
`
		testFile := filepath.Join(subDir, "calendar.yaml")
		err = os.WriteFile(testFile, []byte(validEventSource), 0644)
		require.NoError(t, err)

		// Test directory linting without recursive
		hasErrors := false
		hasWarnings := false
		err = lintDirectory(tmpDir, false, &hasErrors, &hasWarnings)
		assert.NoError(t, err)

		// Test directory linting with recursive
		hasErrors = false
		hasWarnings = false
		err = lintDirectory(tmpDir, true, &hasErrors, &hasWarnings)
		assert.NoError(t, err)
	})

	t.Run("Invalid YAML", func(t *testing.T) {
		invalidYAML := `this is not valid yaml: {{{`
		testFile := filepath.Join(tmpDir, "invalid.yaml")
		err := os.WriteFile(testFile, []byte(invalidYAML), 0644)
		require.NoError(t, err)

		// Test linting invalid YAML
		hasErrors := false
		hasWarnings := false
		err = lintFile(testFile, &hasErrors, &hasWarnings)
		// Should not panic or crash
		assert.NoError(t, err)
	})
}
