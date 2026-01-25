package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/client/clientset/versioned/scheme"
	eventsourcecontroller "github.com/argoproj/argo-events/pkg/reconciler/eventsource"
	sensorcontroller "github.com/argoproj/argo-events/pkg/reconciler/sensor"
)

func NewLintCommand() *cobra.Command {
	var (
		recursive bool
		strict    bool
	)

	command := &cobra.Command{
		Use:   "lint PATH...",
		Short: "Validate EventSource and Sensor resource files",
		Long: `Lint validates EventSource and Sensor resource files.
It performs the same validation checks that are done during resource creation and updates.

Examples:
  # Validate a single file
  argo-events lint event-source.yaml

  # Validate multiple files
  argo-events lint event-source.yaml sensor.yaml

  # Validate all YAML files in a directory
  argo-events lint examples/event-sources/

  # Validate all YAML files in a directory recursively
  argo-events lint -R examples/

  # Validate with strict mode (no warnings allowed)
  argo-events lint --strict event-source.yaml
`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("Error: at least one path must be specified")
				os.Exit(1)
			}

			hasErrors := false
			hasWarnings := false

			for _, path := range args {
				if err := lintPath(path, recursive, &hasErrors, &hasWarnings); err != nil {
					fmt.Fprintf(os.Stderr, "Error processing path %s: %v\n", path, err)
					hasErrors = true
				}
			}

			if hasErrors {
				os.Exit(1)
			}
			if strict && hasWarnings {
				os.Exit(1)
			}
		},
	}

	command.Flags().BoolVarP(&recursive, "recursive", "R", false, "Process directories recursively")
	command.Flags().BoolVar(&strict, "strict", false, "Fail on warnings")

	return command
}

func lintPath(path string, recursive bool, hasErrors *bool, hasWarnings *bool) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return lintDirectory(path, recursive, hasErrors, hasWarnings)
	}

	return lintFile(path, hasErrors, hasWarnings)
}

func lintDirectory(dir string, recursive bool, hasErrors *bool, hasWarnings *bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			if recursive {
				if err := lintDirectory(path, recursive, hasErrors, hasWarnings); err != nil {
					return err
				}
			}
			continue
		}

		// Only process YAML files
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext == ".yaml" || ext == ".yml" {
			if err := lintFile(path, hasErrors, hasWarnings); err != nil {
				// Continue processing other files even if one fails
				*hasErrors = true
				fmt.Fprintf(os.Stderr, "Error processing file %s: %v\n", path, err)
			}
		}
	}

	return nil
}

func lintFile(filename string, hasErrors *bool, hasWarnings *bool) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	// Split the file by "---" to handle multiple YAML documents
	documents := strings.Split(string(data), "\n---\n")

	foundResource := false
	for i, doc := range documents {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		// Try to decode as unstructured to get the kind
		decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(doc), 4096)
		var obj unstructured.Unstructured
		if err := decoder.Decode(&obj); err != nil {
			// Skip non-Kubernetes resources or invalid YAML
			continue
		}

		kind := obj.GetKind()
		if kind != "EventSource" && kind != "Sensor" {
			// Skip resources that are not EventSource or Sensor
			continue
		}

		foundResource = true
		docNum := ""
		if len(documents) > 1 {
			docNum = fmt.Sprintf(" (document %d)", i+1)
		}

		// Now decode it properly using the scheme
		decoder = yaml.NewYAMLOrJSONDecoder(strings.NewReader(doc), 4096)
		runtimeObj, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(doc), nil, nil)
		if err != nil {
			fmt.Printf("✗ %s%s: Failed to decode\n", filename, docNum)
			fmt.Printf("  Error: %v\n", err)
			*hasErrors = true
			continue
		}

		switch resource := runtimeObj.(type) {
		case *v1alpha1.EventSource:
			if err := lintEventSource(filename+docNum, resource, hasErrors, hasWarnings); err != nil {
				return err
			}
		case *v1alpha1.Sensor:
			if err := lintSensor(filename+docNum, resource, hasErrors, hasWarnings); err != nil {
				return err
			}
		default:
			// This shouldn't happen since we already filtered by kind
			continue
		}
	}

	if !foundResource {
		// Don't print anything for files that don't contain EventSource or Sensor
		return nil
	}

	return nil
}

func lintEventSource(filename string, eventSource *v1alpha1.EventSource, hasErrors *bool, hasWarnings *bool) error {
	if err := eventsourcecontroller.ValidateEventSource(eventSource); err != nil {
		fmt.Printf("✗ %s: EventSource validation failed\n", filename)
		fmt.Printf("  Error: %v\n", err)
		*hasErrors = true
	} else {
		fmt.Printf("✓ %s: EventSource is valid\n", filename)
	}
	return nil
}

func lintSensor(filename string, sensor *v1alpha1.Sensor, hasErrors *bool, hasWarnings *bool) error {
	// For sensor validation, we need an EventBus, but for linting purposes
	// we'll create a dummy one or skip EventBus-specific validation
	// We'll validate as much as we can without an actual EventBus
	
	// Create a minimal EventBus for validation
	// We use nil EventBus and check if the validation can proceed
	// The ValidateSensor function will report if EventBus is needed
	dummyEventBus := &v1alpha1.EventBus{
		Spec: v1alpha1.EventBusSpec{
			JetStream: &v1alpha1.JetStreamBus{}, // Provide a minimal valid spec
		},
	}

	if err := sensorcontroller.ValidateSensor(sensor, dummyEventBus); err != nil {
		// Check if the error is about EventBus being nil or missing
		errMsg := err.Error()
		if strings.Contains(errMsg, "eventbus") || strings.Contains(errMsg, "EventBus") {
			fmt.Printf("! %s: Sensor validation incomplete (EventBus not available)\n", filename)
			fmt.Printf("  Note: %v\n", err)
			*hasWarnings = true
		} else {
			fmt.Printf("✗ %s: Sensor validation failed\n", filename)
			fmt.Printf("  Error: %v\n", err)
			*hasErrors = true
		}
	} else {
		fmt.Printf("✓ %s: Sensor is valid\n", filename)
	}
	return nil
}
