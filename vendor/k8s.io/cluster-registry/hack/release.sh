#!/usr/bin/env bash
# Copyright 2017 The Kubernetes Authors.
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

set -euo pipefail

RELEASE_TAG=${1:-}
BUILD_DATE="$(TZ=Etc/UTC date +%Y%m%d)"
RELEASE_VERSION="${RELEASE_TAG:-$BUILD_DATE}"
GCP_PROJECT=${GCP_PROJECT:-crreleases}
GCS_BUCKET=${GCS_BUCKET:-crreleases}
GCR_REPO_PATH="${GCP_PROJECT}/clusterregistry"
SCRIPT_ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
TMPDIR="$(mktemp -d /tmp/crrelXXXXXXXX)"

function clean_up() {
  if [[ "${TMPDIR}" == "/tmp/crrel"* ]]; then
    rm -rf "${TMPDIR}"
  fi
}
trap clean_up EXIT

RELEASE_TMPDIR="${TMPDIR}/${RELEASE_VERSION}"

# Check for and install necessary dependencies.
command -v bazel >/dev/null 2>&1 || { echo >&2 "Please install bazel before running this script."; exit 1; }
command -v gcloud >/dev/null 2>&1 || { echo >&2 "Please install gcloud before running this script."; exit 1; }
gcloud components install gsutil docker-credential-gcr
docker-credential-gcr configure-docker 1>&2

cd "${SCRIPT_ROOT}/.."

# Build the tarballs of the tools.
bazel build \
  //cmd/crinit:clusterregistry-client \
  //cmd/clusterregistry:clusterregistry-server

# Copy the archives.
mkdir -p "${RELEASE_TMPDIR}"
cp \
  bazel-bin/cmd/crinit/clusterregistry-client.tar.gz \
  bazel-bin/cmd/clusterregistry/clusterregistry-server.tar.gz \
  "${RELEASE_TMPDIR}"

# Create the `latest` file.
echo "${RELEASE_VERSION}" > "${TMPDIR}/latest"

pushd "${RELEASE_TMPDIR}" 1>&2

# Create the SHAs.
sha256sum clusterregistry-client.tar.gz > clusterregistry-client.tar.gz.sha
sha256sum clusterregistry-server.tar.gz > clusterregistry-server.tar.gz.sha

popd 1>&2

SUBDIR=""
LATEST_TAG="latest"
if [[ -z "${RELEASE_TAG}" ]]; then
  SUBDIR="nightly/"
  LATEST_TAG="latest_nightly"
fi

# Upload the files to GCS.
gsutil -m cp -r "${TMPDIR}"/* "gs://${GCS_BUCKET}/${SUBDIR}"

# Push the server container image.
bazel run //cmd/clusterregistry:push-clusterregistry-image --define repository="${GCR_REPO_PATH}" 1>&2

# Adjust the tags on the image. The `push-clusterregistry-image` rule tags the
# pushed image with the `dev` tag by default. This consistent tag allows the
# tool to easily add other tags to the image. The tool then removes the `dev`
# tag since this is not a development image.
gcloud container images add-tag --quiet \
  "gcr.io/${GCR_REPO_PATH}:dev" \
  "gcr.io/${GCR_REPO_PATH}:${RELEASE_VERSION}"
gcloud container images add-tag --quiet \
  "gcr.io/${GCR_REPO_PATH}:dev" \
  "gcr.io/${GCR_REPO_PATH}:${LATEST_TAG}"
gcloud container images untag --quiet \
  "gcr.io/${GCR_REPO_PATH}:dev"

# Echo a release note to stdout for later use.
ROOT_GCS_PATH="https://storage.googleapis.com/${GCS_BUCKET}/${SUBDIR}${RELEASE_VERSION}"
cat <<END
# ${RELEASE_TAG}

clusterregistry Docker image: \`gcr.io/${GCR_REPO_PATH}:${RELEASE_VERSION}\`

# Download links

## Client (crinit)
[client](${ROOT_GCS_PATH}/clusterregistry-client.tar.gz)
[client SHA](${ROOT_GCS_PATH}/clusterregistry-client.tar.gz.sha)

## Server (clusterregistry)
[server](${ROOT_GCS_PATH}/clusterregistry-server.tar.gz)
[server SHA](${ROOT_GCS_PATH}/clusterregistry-server.tar.gz.sha)
END
