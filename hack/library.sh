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
  go mod vendor
}	

ensure_mockery() {
  if [ "`command -v mockery`" = "" ]; then
    warning "Please install mockery with - brew install vektra/tap/mockery"
    exit 1
  fi
}

uname_os() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$os" in
    msys*) os="windows" ;;
    mingw*) os="windows" ;;
    cygwin*) os="windows" ;;
    win*) os="windows" ;;
    sunos) [ "$(uname -o)" = "illumos" ] && os=illumos ;;
  esac
  echo "$os"
}

uname_arch() {
  arch=$(uname -m)
  case $arch in
    x86_64) arch="amd64" ;;
    x86) arch="386" ;;
    i686) arch="386" ;;
    i386) arch="386" ;;
    i86pc) arch="amd64" ;;
    aarch64) arch="arm64" ;;
    armv5*) arch="armv5" ;;
    armv6*) arch="armv6" ;;
    armv7*) arch="armv7" ;;
    loongarch64) arch="loong64" ;;
  esac
  echo "${arch}"
}

install-protobuf() {
  local install_dir=""
  while [[ "$#" -gt 0 ]]; do
    case "$1" in
      "--install-dir")
        install_dir="$2"
        shift 2
        ;;
      *)
        if [[ "$1" =~ ^-- ]]; then
          echo "unknown argument: $1" >&2
          return 1
        fi
        if [ -n "$install_dir" ]; then
          echo "too many arguments: $1 (already have $install_dir)" >&2
          return 1
        fi
        install_dir="$1"
        shift
        ;;
    esac
  done

  if [[ -z "${install_dir}" ]]; then
    echo "install-dir argument is required" >&2
    return 1
  fi

  if [[ ! -d "${install_dir}" ]]; then
    echo "${install_dir} does not exist" >&2
    return 1
  fi

  # protobuf version
  local protobuf_version=27.2
  local pb_rel="https://github.com/protocolbuffers/protobuf/releases"
  local os=$(uname_os)
  local arch=$(uname_arch)

  echo "OS: $os  ARCH: $arch"
  if [[ "$arch" = "amd64" ]]; then
    arch="x86_64"
  elif [[ "$arch" = "arm64" ]]; then
    arch="aarch_64"
  fi
  local binary_url=${pb_rel}/download/v${protobuf_version}/protoc-${protobuf_version}-${os}-${arch}.zip
  if [[ "$os" = "darwin" ]]; then
    binary_url=${pb_rel}/download/v${protobuf_version}/protoc-${protobuf_version}-osx-universal_binary.zip
  fi
  echo "Downloading $binary_url"

  tmp=$(mktemp -d)
  curl -sL -o ${tmp}/protoc-${protobuf_version}-${os}-${arch}.zip $binary_url
  unzip ${tmp}/protoc-${protobuf_version}-${os}-${arch}.zip -d ${install_dir}
  rm -rf ${tmp}
}

