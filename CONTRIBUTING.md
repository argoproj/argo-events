# Contributing

## Report a Bug
Open an issue. Please include descriptions of the following:
- Observations
- Expectations
- Steps to reproduce

## Contribute a Bug Fix
- Report the bug first
- Create a pull request for the fix

## Suggest a New Feature
- Create a new issue to start a discussion around new topic. Label the issue as `new-feature`

## Setup your DEV environment

### Requirements
- Golang 1.10
- Docker
- dep

### Quickstart
See the [quickstart guide](./docs/quickstart.md) for help in getting started.

## Changing Types
If you're making a change to the `pkg/apis/sensor/v1alpha1` package, please ensure you re-run the K8 code-generator scripts found in the `/hack` folder. First, ensure you have the `generate-groups.sh` script at the path: `vendor/k8s.io/code-generator/`. Next run the following commands in order:
```
$ make codegen
```
