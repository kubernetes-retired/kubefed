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

# This script automates the build and testing for the release of a new version
# of kubefed.

set -o errexit
set -o nounset
set -o pipefail

source "$(dirname "${BASH_SOURCE}")/util.sh"
ROOT_DIR="$(cd "$(dirname "$0")/.." ; pwd)"
QUAY_REPO="ifont/kubefed"
GITHUB_REPO="font/kubefed"

# Arguments
RELEASE_TAG="${1-}"

# TODO(font): make util function
function log() {
  echo "##### ${1}..."
}

function verify-command-installed() {
  if ! util::command-installed travis; then
    echo "travis command not found. Please add travis to your PATH and try again." >&2
    return 1
  fi
}

function build-release-artifacts() {
  ${ROOT_DIR}/scripts/build-release-artifacts.sh ${RELEASE_TAG}
}

function create-and-push-tag() {
  git tag -a "${RELEASE_TAG}" -m "Creating release tag ${RELEASE_TAG}"
  git push origin ${RELEASE_TAG}
}

function travis-build-status() {
  local status="$(travis branches --repo ${GITHUB_REPO} | grep "${RELEASE_TAG}" | awk '{print $3}')"
  echo "${status}"
}

function verify-travis-build-started() {
  local buildStatus="$(travis-build-status)"
  [[ "${buildStatus}" == "started" ]]
}

function verify-travis-build-completed() {
  local statuses="${passedStatus} failed errored canceled"
  local buildStatus="$(travis-build-status)"

  for status in ${statuses}; do
    if [[ "${status}" == "${buildStatus}" ]]; then
      return 0
    fi
  done

  return 1
}

function verify-continuous-integration() {
  local passedStatus="passed"
  util::wait-for-condition "kubefed CI build to start" "verify-travis-build-started started" 150
  util::wait-for-condition "kubefed CI build to complete" "verify-travis-build-completed" 3600
  local buildStatus="$(travis-build-status)"

  if [[ "${buildStatus}" == "${passedStatus}" ]]; then
    echo "kubefed CI build ${passedStatus}"
  else
    echo "kubefed CI build ${buildStatus}. Exiting."
    exit 1
  fi
}

function quay-image-status() {
  local quayImagesApi="https://quay.io/api/v1/repository/${QUAY_REPO}/image/"
  curl -s ${quayImagesApi} | grep "${RELEASE_TAG}" &> /dev/null
}

function verify-container-image() {
  util::wait-for-condition "kubefed container image in quay" "quay-image-status" 60
}

function update-changelog() {
  sed -i "/# Unreleased/a \\\n# ${RELEASE_TAG}" CHANGELOG.md
}

RELEASE_TAG_REGEX="^v[0-9]+\.[0-9]+\.[0-9]+(-rc[0-9]+)?$"
if [[ ! "${RELEASE_TAG}" =~ ${RELEASE_TAG_REGEX} ]]; then
  >&2 echo "usage: $0 <release tag of the form v[0-9]+.[0-9]+.[0-9]+(-rc[0-9]+)?>"
  exit 1
fi

log "Verifying travis CLI command installed"
verify-command-installed

log "Building release artifacts first to make sure build succeeds"
build-release-artifacts

log "Creating local git annotated tag and pushing tag to kick off build process"
create-and-push-tag

log "Verifing image builds and completes successfully in Travis"
verify-continuous-integration

log "Verifying container image tags and pushes successfully to Quay"
verify-container-image

log "Updating CHANGELOG.md"
update-changelog

# TODO(font): Consider making the next step to create a github release
# interactive to allow for curating the CHANGELOG while maximizing automation
# as much as possible.
echo -e "\nThe kubefed version ${RELEASE_TAG} is now ready to be released.\nPlease update CHANGELOG.md for correct wording and run ${ROOT_DIR}/scripts/create-gh-release.sh when ready."
