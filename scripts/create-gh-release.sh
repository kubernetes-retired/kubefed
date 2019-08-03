#!/usr/bin/env bash

# Copyright 2019 The Kubernetes Authors.
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

source "$(dirname "${BASH_SOURCE}")/util.sh"
ROOT_DIR="$(cd "$(dirname "$0")/.." ; pwd)"

# Arguments
RELEASE_TAG="${1-}"

# TODO(font): make util function
function log() {
  echo "##### ${1}..."
}

function verify-command-installed() {
  if ! util::command-installed hub; then
    echo "hub command not found. Please add hub to your PATH and try again." >&2
    return 1
  fi
}

function prime-command-for-auth() {
  # Run valid but benign hub command to prompt user for github auth
  hub release show -f "%n" v0.1.0-rc5
}

function github-release-template() {
  # Add leading # for markdown heading level 1 (h1)
  local regex="$(echo ${RELEASE_TAG_REGEX/^/^\# })"
  cat <<EOF
${RELEASE_TAG}

## Changelog

$(sed -E "1,/${regex}/d ; /${regex}/,\$d" CHANGELOG.md)

## Artifacts

### kubefedctl, command line tool to join clusters, enable type federation, and convert resources to their federated equivalents
See asset links below for \`kubefedctl-x.x.x-<os>-<arch>.tgz\`

### Helm chart, to deploy federation as per user guide instructions
See asset link below for \`kubefed-x.x.x.tgz\`

### Controller-manager image
**_quay.io/kubernetes-multicluster/kubefed:${RELEASE_TAG}_**

### User Guide
[**User Guide**](https://github.com/kubernetes-sigs/kubefed/blob/master/docs/userguide.md)
EOF
}

function create-github-release() {
  # Output github release template with the RELEASE_TAG and the CHANGELOG
  # details.
  local releaseFile="kubefed-${RELEASE_TAG}-github-release.md"
  github-release-template > "${releaseFile}"

  # Build asset attach arguments for hub release command
  local assetArgs
  for asset in $(cat kubefed-${RELEASE_TAG}-asset-files.txt); do
    assetArgs+="--attach ${asset} "
  done

  # Remove trailing whitespace
  assetArgs="$(echo ${assetArgs})"

  # TODO(font): Add draft and prerelease options to this script.
  hub release create --draft --prerelease ${assetArgs} -F "${releaseFile}" "${RELEASE_TAG}"
}

# TODO(font): Consider creating a PR and either having this script auto-merge
# it, or leaving it up to the user for merging. Also consider performing this
# step BEFORE tagging and pushing a new release so that the release tag
# contains all the necessary artifacts.
function update-master-branch() {
  # Assumes we're already on master and pushing to the correct remote.
  local commitFiles="${ROOT_DIR}/CHANGELOG.md ${ROOT_DIR}/charts/index.yaml"
  git commit ${commitFiles} -m "Update repo for release ${RELEASE_TAG}"
  git push
}

RELEASE_TAG_REGEX="^v[0-9]+\.[0-9]+\.[0-9]+(-rc[0-9]+)?$"
if [[ ! "${RELEASE_TAG}" =~ ${RELEASE_TAG_REGEX} ]]; then
  >&2 echo "usage: $0 <release tag of the form v[0-9]+.[0-9]+.[0-9]+(-rc[0-9]+)?>"
  exit 1
fi

log "Verifying hub CLI command installed"
verify-command-installed

log "Priming hub CLI command for authentication"
prime-command-for-auth

log "Creating github release"
create-github-release

log "Updating master branch with release changes"
update-master-branch

echo -e "\nCreated kubefed ${RELEASE_TAG} github draft release. Go publish at the following URL when ready:"
hub release show --format "%U%n" "${RELEASE_TAG}"
