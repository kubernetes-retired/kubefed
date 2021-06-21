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

# This script automates the build and testing of the release of a new version
# of kubefed.

set -o errexit
set -o nounset
set -o pipefail

source "$(dirname "${BASH_SOURCE}")/util.sh"
ROOT_DIR="$(cd "$(dirname "$0")/.." ; pwd)"

# Arguments
RELEASE_TAG="${1-}"

# Globals
QUAY_REPO="${QUAY_REPO:-kubernetes-multicluster/kubefed}"
GITHUB_REPO="${GITHUB_REPO:-kubernetes-sigs/kubefed}"
GITHUB_REMOTE_UPSTREAM_NAME="${GITHUB_REMOTE_UPSTREAM_NAME:-upstream}"

function verify-command-installed() {
  if ! util::command-installed gh; then
    echo "gh command not found. Please add gh to your PATH and try again." >&2
    return 1
  fi
}

function build-release-artifacts() {
  "${ROOT_DIR}/scripts/build-release-artifacts.sh" "${RELEASE_TAG}"
}

function create-and-push-tag() {
  # Use the upstream git remote convention name used by hub.

  if git ls-remote --tags "${GITHUB_REMOTE_UPSTREAM_NAME}" refs/tags/"${RELEASE_TAG}" | grep -E refs/tags/"${RELEASE_TAG}" &> /dev/null; then
    echo "git tag ${RELEASE_TAG} already exists in ${GITHUB_REMOTE_UPSTREAM_NAME} remote. Continuing..."
    return 0
  fi

  # Make sure upstream is updated with the latest commit.
  git fetch "${GITHUB_REMOTE_UPSTREAM_NAME}" --prune

  # This creates an annotated tag required to ensure that the KubeFed binaries
  # are versioned correctly.
  git tag -s -a "${RELEASE_TAG}" "${GITHUB_REMOTE_UPSTREAM_NAME}/master" -m "Creating release tag ${RELEASE_TAG}"
  git push "${GITHUB_REMOTE_UPSTREAM_NAME}" "${RELEASE_TAG}"
}

function build-status() {
  local buildStatusAPIURL="/repos/${GITHUB_REPO}/actions/runs?per_page=1&event=push&branch=${RELEASE_TAG}"
  gh api --jq '.workflow_runs[0].status' "${buildStatusAPIURL}"
}

function build-conclusion() {
  local buildStatusAPIURL="/repos/${GITHUB_REPO}/actions/runs?per_page=1&event=push&branch=${RELEASE_TAG}"
  gh api --jq '.workflow_runs[0].conclusion' "${buildStatusAPIURL}"
}

function build-started() {
  if [[ "$(build-status)" == "in_progress" ]]; then
    return 0
  fi
  return 1
}

function build-completed() {
  if [[ "$(build-status)" == "completed" ]]; then
    return 0
  fi
  return 1
}

function verify-continuous-integration() {
  util::wait-for-condition "kubefed CI build to start" "build-started" 1200
  util::wait-for-condition "kubefed CI build to complete" "build-completed" 3600

  local buildConclusion="$(build-conclusion)"

  if [[ "${buildConclusion}" == "success" ]]; then
    echo "kubefed CI build success"
  else
    echo "kubefed CI build ${buildConclusion}. Exiting."
    exit 1
  fi
}

function quay-image-status() {
  local quayImagesAPIURL="https://quay.io/api/v1/repository/${QUAY_REPO}/tag/${RELEASE_TAG}/images"
  curl -fsSL "${quayImagesAPIURL}" &> /dev/null
}

function verify-container-image() {
  util::wait-for-condition "kubefed container image in quay" "quay-image-status" 60
}

function update-changelog() {
  if ! grep "^# ${RELEASE_TAG}" CHANGELOG.md &> /dev/null; then
    sed -i "/# Unreleased/a \\\n# ${RELEASE_TAG}" CHANGELOG.md
  fi
}

if [[ ! "${RELEASE_TAG}" =~ ${RELEASE_TAG_REGEX} ]]; then
  >&2 echo "usage: $0 <release tag of the form v[0-9]+.[0-9]+.[0-9]+(-(alpha|beta|rc)\.?[0-9]+)?>"
  exit 1
fi

util::log "Verifying gh CLI command installed"
verify-command-installed

util::log "Building release artifacts first to make sure build succeeds"
build-release-artifacts

util::log "Creating local git signed and annotated tag and pushing tag to kick off build process"
create-and-push-tag

util::log "Verifying image builds and completes successfully in Github Actions. This can take a while (~1 hour)"
verify-continuous-integration

util::log "Verifying container image tags and pushes successfully to Quay"
verify-container-image

util::log "Updating CHANGELOG.md"
update-changelog

# TODO(font): Consider making the next step to create a github release
# interactive to allow for curating the CHANGELOG while maximizing automation
# as much as possible.
echo -e "\nThe kubefed version ${RELEASE_TAG} is now ready to be released.\nPlease update CHANGELOG.md for correct wording and run ${ROOT_DIR}/scripts/create-gh-release.sh when ready."
