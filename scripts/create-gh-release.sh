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

source "$(dirname "${BASH_SOURCE[0]}")/util.sh"
ROOT_DIR="$(cd "$(dirname "$0")/.." ; pwd)"

# Arguments
RELEASE_TAG="${1-}"

# Globals
GITHUB_REMOTE_FORK_NAME="${GITHUB_REMOTE_FORK_NAME:-origin}"
GITHUB_REMOTE_UPSTREAM_NAME="${GITHUB_REMOTE_UPSTREAM_NAME:-upstream}"
GITHUB_PR_BASE_BRANCH="${GITHUB_PR_BASE_BRANCH:-master}"

function verify-command-installed() {
  if ! util::command-installed gh; then
    echo "gh command not found. Please add gh to your PATH and try again." >&2
    return 1
  fi
}

function prime-command-for-auth() {
  gh auth status
}

RELEASE_ASSETS_FILE="kubefed-${RELEASE_TAG}-asset-files.txt"

function verify-assets-file-exists() {
  if [[ ! -f "${RELEASE_ASSETS_FILE}" ]]; then
    echo "ERROR: kubefed release assets file '${RELEASE_ASSETS_FILE}' does not exist. Please run ${ROOT_DIR}/scripts/build-release-artifacts.sh and try again."
    return 1
  fi
}

function github-release-template() {
  # Add leading # for markdown heading level 1 (h1)
  local regex="${RELEASE_TAG_REGEX/^/^\# }"
  cat <<EOF
$(sed -En "/^# ${RELEASE_TAG}/,/${regex}/p" CHANGELOG.md | head -n-2)

## Artifacts

### kubefedctl, command line tool to join clusters, enable type federation, and convert resources to their federated equivalents
See asset links below for \`kubefedctl-x.x.x-<os>-<arch>.tgz\`

### Helm chart, to deploy federation as per user guide instructions
See asset link below for \`kubefed-x.x.x.tgz\`

### Controller-manager image
**_docker.io/mesosphere/kubefed:${RELEASE_TAG}_**

### User Guide
[**User Guide**](https://github.com/kubernetes-sigs/kubefed/blob/master/docs/userguide.md)
EOF
}

function create-github-release() {
  # Output github release template with the RELEASE_TAG and the CHANGELOG
  # details.
  local releaseFile="kubefed-${RELEASE_TAG}-github-release.md"
  github-release-template > "${releaseFile}"

  # Build asset attach arguments for gh release command
  local assetFiles=()
  for asset in $(cat "${RELEASE_ASSETS_FILE}"); do
    assetFiles+=("${asset}")
  done

  # TODO(font): Add draft and prerelease options to this script.
  gh release create --draft --prerelease -F "${releaseFile}" "${RELEASE_TAG}" "${assetFiles[@]}"
}

# TODO(font): Consider performing this step BEFORE tagging and pushing a new
# release so that the release tag contains all the necessary artifacts.
function create-release-pr() {
  local commitFiles=("${ROOT_DIR}/CHANGELOG.md" "${ROOT_DIR}/charts/index.yaml")

  # Use the origin and upstream git remote convention names used by gh.
  git checkout -b "${RELEASE_TAG}-rel" --no-track "${GITHUB_REMOTE_UPSTREAM_NAME}/master"
  git commit "${commitFiles[@]}" -m "Update repo for release ${RELEASE_TAG}"
  git push --set-upstream "${GITHUB_REMOTE_FORK_NAME}" "${RELEASE_TAG}-rel"
  PR_URL="$(gh pr create --base "${GITHUB_PR_BASE_BRANCH}" --title "Update repo for release ${RELEASE_TAG}" -b "" --fill)"

}

if [[ ! "${RELEASE_TAG}" =~ ${RELEASE_TAG_REGEX} ]]; then
  >&2 echo "usage: $0 <release tag of the form v[0-9]+.[0-9]+.[0-9]+(-(alpha|beta|rc)\.?[0-9]+)?>"
  exit 1
fi

util::log "Verifying gh CLI command installed"
verify-command-installed

util::log "Priming gh CLI command for authentication"
prime-command-for-auth

util::log "Verifying release assets file exists"
verify-assets-file-exists

util::log "Creating pull request with release ${RELEASE_TAG} changes"
create-release-pr

util::log "Creating github release"
create-github-release

echo -e "\nCreated kubefed ${RELEASE_TAG} pull request and github draft release. Go merge and publish at the following URLs when ready:"
echo "${PR_URL}"
gh release view --json url --jq .url "${RELEASE_TAG}"
