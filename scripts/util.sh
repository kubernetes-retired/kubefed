#!/usr/bin/env bash

# Copyright 2018 The Kubernetes Authors.
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

# This library holds common bash variables and utility functions.

# Variables
#
#
# RELEASE_TAG_REGEX contains the regular expression used to validate a semantic
# version.
export RELEASE_TAG_REGEX="^v[0-9]+\.[0-9]+\.[0-9]+(-rc[0-9]+)?$"


# Utility Functions
#
#
# util::command-installed checks if the command from argument 1 is installed.
#
# Globals:
#  None
# Arguments:
#  - 1: command name to check if it is installed in PATH
# Returns:
#  0 if command is installed in PATH
#  1 if the command is NOT installed in PATH
function util::command-installed() {
  command -v "${1}" >/dev/null 2>&1 || return 1
  return 0
}
readonly -f util::command-installed

# util::log echoes the supplied argument with a common header.
#
# Globals:
#  None
# Arguments:
#  - 1: string to echo
# Returns:
#  0
function util::log() {
  echo "##### ${1}..."
}
readonly -f util::log

# util::wait-for-condition blocks until the provided condition becomes true
#
# Globals:
#  None
# Arguments:
#  - 1: message indicating what conditions is being waited for (e.g. 'config to be written')
#  - 2: a string representing an eval'able condition.  When eval'd it should not output
#       anything to stdout or stderr.
#  - 3: optional timeout in seconds.  If not provided, waits forever.
# Returns:
#  1 if the condition is not met before the timeout
function util::wait-for-condition() {
  local msg=$1
  # condition should be a string that can be eval'd.
  local condition=$2
  local timeout=${3:-}

  local start_msg="Waiting for ${msg}"
  local error_msg="[ERROR] Timeout waiting for ${msg}"

  local counter=0
  while ! eval ${condition}; do
    if [[ "${counter}" = "0" ]]; then
      echo -n "${start_msg}"
    fi

    if [[ -z "${timeout}" || "${counter}" -lt "${timeout}" ]]; then
      counter=$((counter + 1))
      if [[ -n "${timeout}" ]]; then
        echo -n '.'
      fi
      sleep 1
    else
      echo -e "\n${error_msg}"
      return 1
    fi
  done

  if [[ "${counter}" != "0" && -n "${timeout}" ]]; then
    echo ' done'
  fi
}
readonly -f util::wait-for-condition
