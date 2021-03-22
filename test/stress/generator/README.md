# Events Generator

A simple tool to generate messages for different event resources.

## Usage

```shell
go run test/stress/generator/main.go --help
Events generator for stress testing.

Usage:
  go run ./test/stress/generator/main.go [flags]
  go [command]

Available Commands:
  help        Help about any command
  sqs         Generate SQS messages
  webhook     Generate webhook event source messages

Flags:
  -d, --duration duration   How long it will run, e.g. 5m, 60s (default 5m0s)
  -h, --help                help for go
      --rps int             Requests per second
      --total int           Total requests

Use "go [command] --help" for more information about a command.

```

### Supported EventSources

Currently only the most common event sources are supported, you are welcome to
contribute the implementation.
