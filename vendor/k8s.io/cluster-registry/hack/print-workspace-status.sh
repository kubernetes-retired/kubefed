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

# This command is used by bazel as the workspace_status_command
# to implement build stamping with git information. It pulls information
# from tags named like vX.Y.Z with an optional postfix, e.g., v1.2.3-alpha.

set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
GIT=(git --work-tree "${REPO_ROOT}")

if GIT_COMMIT=$("${GIT[@]}" rev-parse "HEAD^{commit}" 2>/dev/null); then
  # Check if the tree is dirty. Default to dirty
  GIT_TREE_STATE="dirty"
  if git_status=$("${git[@]}" status --porcelain 2>/dev/null) && [[ -z ${git_status} ]]; then
    GIT_TREE_STATE="clean"
  fi

  # Use git describe to find the version based on annotated tags.
  if GIT_VERSION=$("${GIT[@]}" describe --abbrev=14 "${GIT_COMMIT}^{commit}" 2>/dev/null); then
    # This translates the "git describe" to an actual semver.org
    # compatible semantic version that looks something like this:
    #   v1.1.0-alpha.0.6+84c76d1142ea4d
    DASHES_IN_VERSION=$(echo "${GIT_VERSION}" | sed "s/[^-]//g")
    if [[ "${DASHES_IN_VERSION}" == "---" ]] ; then
      # We have distance to subversion (v1.1.0-subversion-1-gCommitHash)
      GIT_VERSION=$(echo "${GIT_VERSION}" | sed "s/-\([0-9]\{1,\}\)-g\([0-9a-f]\{14\}\)$/.\1\+\2/")
    elif [[ "${DASHES_IN_VERSION}" == "--" ]] ; then
      # We have distance to base tag (v1.1.0-1-gCommitHash)
      GIT_VERSION=$(echo "${GIT_VERSION}" | sed "s/-g\([0-9a-f]\{14\}\)$/+\1/")
    fi
    if [[ "${GIT_TREE_STATE}" == "dirty" ]]; then
      # git describe --dirty only considers changes to existing files, but
      # that is problematic since new untracked .go files affect the build,
      # so use our idea of "dirty" from git status instead.
      GIT_VERSION+="-dirty"
    fi
  fi
fi

# Set default version information if there are no tags to read.
if [[ -z ${GIT_VERSION-} ]]; then
  GIT_VERSION="0.0"
fi

# Prefix with STABLE_ so that these values are saved to stable-status.txt
# instead of volatile-status.txt.
# Stamped rules will be retriggered by changes to stable-status.txt, but not by
# changes to volatile-status.txt.
# IMPORTANT: the camelCase vars should match the list in pkg/version/def.bzl.
cat <<EOF
STABLE_BUILD_GIT_COMMIT ${GIT_COMMIT}
STABLE_BUILD_SCM_STATUS ${GIT_TREE_STATE}
STABLE_BUILD_SCM_REVISION ${GIT_VERSION}
buildDate $(date -u +'%Y-%m-%dT%H:%M:%SZ')
gitCommit ${GIT_COMMIT}
gitTreeState ${GIT_TREE_STATE}
semanticVersion ${GIT_VERSION}
EOF
