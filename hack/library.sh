#!/bin/bash

readonly REPO_ROOT="$(git rev-parse --show-toplevel)"

# Display a box banner.
# Parameters: $1 - character to use for the box.
#             $2 - banner message.
function make_banner() {
  local msg="$1$1$1$1 $2 $1$1$1$1"
  local border="${msg//[-0-9A-Za-z _.,:\/()]/$1}"
  echo -e "${border}\n${msg}\n${border}"
}

# Simple header for logging purposes.
function header() {
  local upper="$(echo $1 | tr a-z A-Z)"
  make_banner "+" "${upper}"
}

# Simple subheader for logging purposes.
function subheader() {
  make_banner "-" "$1"
}

# Simple warning banner for logging purposes.
function warning() {
  make_banner "!" "$1"
}

function make_fake_paths() {
  FAKE_GOPATH="$(mktemp -d)"
  trap 'rm -rf ${FAKE_GOPATH}' EXIT
  FAKE_REPOPATH="${FAKE_GOPATH}/src/github.com/argoproj/argo-events"
  mkdir -p "$(dirname "${FAKE_REPOPATH}")" && ln -s "${REPO_ROOT}" "${FAKE_REPOPATH}"
}

ensure_vendor() {
  if [ ! -d "${REPO_ROOT}/vendor" ]; then
    go mod vendor
  fi
}

ensure_pandoc() {
  if [ "`command -v pandoc`" = "" ]; then
    warning "Please install pandoc with - brew install pandoc"
    exit 1
  fi
}

ensure_mockery() {
  if [ "`command -v mockery`" = "" ]; then
    warning "Please install mockery with - brew install vektra/tap/mockery"
    exit 1
  fi
}

ensure_protobuf() {
  if [ "`command -v protoc`" = "" ]; then
    warning "Please install protobuf with - brew install protobuf"
    exit 1
  fi
}

