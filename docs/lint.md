# Linting EventSource and Sensor Resources

The `argo-events lint` command validates EventSource and Sensor resource files without deploying them to a cluster. This allows you to catch configuration errors early in your development process and in CI/CD pipelines.

## Overview

The lint command performs the same validation checks that the Argo Events controllers run when resources are created or updated. This includes:

- **EventSource validation**: Validates event source configuration, required fields, and type-specific requirements
- **Sensor validation**: Validates sensor dependencies, triggers, and trigger templates
- **Schema validation**: Ensures resources conform to the Kubernetes API schema

## Installation

The lint command is built into the `argo-events` binary. After building or installing argo-events, the lint command is available:

```bash
# Build from source
make build

# Use the binary
./dist/argo-events lint --help
```

## Basic Usage

### Validate a single file

```bash
argo-events lint event-source.yaml
```

Output:
```
✓ event-source.yaml: EventSource is valid
```

### Validate multiple files

```bash
argo-events lint event-source.yaml sensor.yaml
```

Output:
```
✓ event-source.yaml: EventSource is valid
✓ sensor.yaml: Sensor is valid
```

### Validate a directory

```bash
argo-events lint examples/event-sources/
```

Output:
```
✓ examples/event-sources/calendar.yaml: EventSource is valid
✓ examples/event-sources/webhook.yaml: EventSource is valid
✓ examples/event-sources/kafka.yaml: EventSource is valid
...
```

### Validate directories recursively

```bash
argo-events lint -R examples/
```

This will validate all `.yaml` and `.yml` files in the `examples/` directory and all subdirectories.

## Flags

### `-R, --recursive`

Process directories recursively.

```bash
argo-events lint -R examples/
```

### `--strict`

Fail on warnings in addition to errors.

```bash
argo-events lint --strict sensor.yaml
```

## Exit Codes

The lint command uses exit codes to indicate success or failure, making it suitable for use in CI/CD pipelines:

- **0**: All resources are valid
- **1**: One or more resources have validation errors

## Examples

### CI/CD Integration

#### GitHub Actions

```yaml
name: Lint Argo Events Resources
on: [push, pull_request]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.20'
      
      - name: Install argo-events
        run: |
          git clone https://github.com/argoproj/argo-events.git
          cd argo-events
          make build
          sudo mv dist/argo-events /usr/local/bin/
      
      - name: Lint EventSources and Sensors
        run: argo-events lint -R manifests/
```

#### GitLab CI

```yaml
lint:
  stage: test
  image: golang:1.20
  script:
    - git clone https://github.com/argoproj/argo-events.git
    - cd argo-events && make build
    - ./dist/argo-events lint -R ../manifests/
```

#### Pre-commit Hook

Create `.git/hooks/pre-commit`:

```bash
#!/bin/bash
# Lint all EventSource and Sensor files before committing

FILES=$(git diff --cached --name-only --diff-filter=ACM | grep -E '\.(yaml|yml)$')

if [ -z "$FILES" ]; then
  exit 0
fi

if ! command -v argo-events &> /dev/null; then
  echo "argo-events not found. Please install it first."
  exit 1
fi

echo "Linting Argo Events resources..."
argo-events lint $FILES

if [ $? -ne 0 ]; then
  echo "Linting failed. Please fix the errors before committing."
  exit 1
fi

echo "All resources are valid!"
exit 0
```

### Validate Multi-Document YAML Files

The lint command supports YAML files with multiple documents separated by `---`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: webhook
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
  name: webhook
spec:
  dependencies:
    - name: test-dep
      eventSourceName: webhook
      eventName: example
  triggers:
    - template:
        name: webhook-workflow-trigger
        k8s:
          operation: create
          source:
            resource:
              apiVersion: v1
              kind: ConfigMap
              metadata:
                name: example
              data:
                key: value
```

```bash
argo-events lint combined.yaml
```

Output:
```
✓ combined.yaml (document 1): EventSource is valid
✓ combined.yaml (document 2): Sensor is valid
```

## Common Validation Errors

### EventSource Errors

```bash
✗ webhook.yaml: EventSource validation failed
  Error: endpoint can't be empty
```

**Solution**: Ensure all required fields are specified in the event source configuration.

### Sensor Errors

```bash
✗ sensor.yaml: Sensor validation failed
  Error: no triggers found
```

**Solution**: Add at least one trigger to the sensor spec.

### EventBus Warnings

```bash
! sensor.yaml: Sensor validation incomplete (EventBus not available)
  Note: nil eventbus
```

**Note**: When linting sensors without access to a cluster, the EventBus cannot be validated. This is expected behavior for offline linting and results in a warning rather than an error.

## Ignored Files

The lint command only processes:
- Files with `.yaml` or `.yml` extensions
- Files containing EventSource or Sensor resources (other Kubernetes resources are ignored)

## Limitations

- **EventBus validation**: Sensors require an EventBus to function, but the lint command cannot validate EventBus availability when running offline. A dummy EventBus is used for basic validation.
- **External dependencies**: The lint command does not validate external dependencies like service accounts, secrets, or workflow templates referenced in trigger specifications.

## Troubleshooting

### No output produced

If lint produces no output, ensure:
1. The file contains valid YAML
2. The resources have `kind: EventSource` or `kind: Sensor`
3. The files have `.yaml` or `.yml` extensions

### Permission denied

If you get "permission denied" errors:
```bash
chmod +x dist/argo-events
```

## Related Commands

- Argo Workflows lint: `argo lint workflow.yaml`
- Kubernetes dry-run: `kubectl apply --dry-run=client -f resource.yaml`
- Helm lint: `helm lint chart/`

## Contributing

To improve the lint command or report issues, see [CONTRIBUTING.md](CONTRIBUTING.md).
