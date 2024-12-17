#!/usr/bin/env bash

# Copyright 2024 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

readonly SCRIPT_ROOT="$(cd "$(dirname "${BASH_SOURCE}")"/.. && pwd)"

# Keep outer module cache so we don't need to redownload them each time.
# The build cache already is persisted.
readonly GOMODCACHE="$(go env GOMODCACHE)"
readonly GO111MODULE="on"
readonly GOFLAGS="-mod=readonly"
readonly GOPATH="$(mktemp -d)"
readonly MIN_REQUIRED_GO_VER="$(go list -m -f '{{.GoVersion}}')"

function go_version_matches {
    go version | perl -ne "exit 1 unless m{go version go([0-9]+.[0-9]+)}; exit 1 if (\$1 < ${MIN_REQUIRED_GO_VER})"
    return $?
}

if ! go_version_matches; then
    echo "Go v${MIN_REQUIRED_GO_VER} or later is required to run code generation"
    exit 1
fi

export GOMODCACHE GO111MODULE GOFLAGS GOPATH

readonly APIS_PKG=sigs.k8s.io/cni-dra-driver
readonly VERSIONS=(v1alpha1)

INPUT_DIRS_SPACE=""
for VERSION in "${VERSIONS[@]}"
do
  INPUT_DIRS_SPACE+="${APIS_PKG}/apis/${VERSION} "
done
INPUT_DIRS_SPACE="${INPUT_DIRS_SPACE%,}" # drop trailing space

if [[ "${VERIFY_CODEGEN:-}" == "true" ]]; then
  echo "Running in verification mode"
  readonly VERIFY_FLAG="--verify-only"
fi

readonly COMMON_FLAGS="${VERIFY_FLAG:-} --go-header-file ${SCRIPT_ROOT}/hack/boilerplate/boilerplate.generatego.txt"

echo "Generating ${VERSION} register at ${APIS_PKG}/apis/${VERSION}"
go run k8s.io/code-generator/cmd/register-gen \
    --output-file zz_generated.register.go \
    ${COMMON_FLAGS} \
    ${INPUT_DIRS_SPACE}

for VERSION in "${VERSIONS[@]}"
do
    echo "Generating ${VERSION} deepcopy at ${APIS_PKG}/apis/${VERSION}"
    go run sigs.k8s.io/controller-tools/cmd/controller-gen \
        object:headerFile=${SCRIPT_ROOT}/hack/boilerplate/boilerplate.generatego.txt \
        paths="${APIS_PKG}/apis/${VERSION}"
done
