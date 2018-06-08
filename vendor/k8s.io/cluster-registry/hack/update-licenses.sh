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

# Generates a table of the licenses of the cluster-registry and its vendored
# dependencies, and outputs it to a LICENSES file.
#
# Usage:
#    $0 [--verify-only]

set -euo pipefail

export LANG=C
export LC_ALL=C

###############################################################################
# Process package content
#
# @param package  The incoming package name
# @param type     The type of content (LICENSE or COPYRIGHT)
#
process_content () {
  local package=$1
  local type=$2

  local package_root
  local ensure_pattern
  local dir_root
  local find_maxdepth
  local find_names
  local -a local_files=()

  # Necessary to expand {}
  case ${type} in
      LICENSE) find_names=(-iname 'licen[sc]e*')
               find_maxdepth=1
               # Sadly inconsistent in the wild, but mostly license files
               # containing copyrights, but no readme/notice files containing
               # licenses (except to "see license file")
               ensure_pattern="license|copyright"
               ;;
    # We search READMEs for copyrights and this includes notice files as well
    # Look in as many places as we find files matching
    COPYRIGHT) find_names=(-iname 'notice*' -o -iname 'readme*')
               find_maxdepth=3
               ensure_pattern="copyright"
               ;;
  esac

  # Start search at package root
  case ${package} in
    github.com/*|golang.org/*|bitbucket.org/*)
     package_root=$(echo ${package} |awk -F/ '{ print $1"/"$2"/"$3 }')
     ;;
    go4.org/*)
     package_root=$(echo ${package} |awk -F/ '{ print $1 }')
     ;;
    *)
     package_root=$(echo ${package} |awk -F/ '{ print $1"/"$2 }')
     ;;
  esac

  # Find files - only root and package level
  local_files=($(
    for dir_root in ${package} ${package_root}; do
      [[ -d ${DEPS_DIR}/${dir_root} ]] || continue

      # One (set) of these is fine
      find ${DEPS_DIR}/${dir_root} \
          -xdev -follow -maxdepth ${find_maxdepth} \
          -type f "${find_names[@]}"
    done | sort -u))

  local key
  local f
  key="${package}-${type}"
  if [[ -z "${CONTENT[${key}]-}" ]]; then
    for f in ${local_files[@]-}; do
      # Find some copyright info in any file and break
      if egrep -i -wq "${ensure_pattern}" "${f}"; then
        CONTENT[${key}]="${f}"
        break
      fi
    done
  fi
}


#############################################################################
# MAIN
#############################################################################

# Check bash version
if ((${BASH_VERSINFO[0]}<4)); then
  echo
  echo "ERROR: Bash v4+ required."
  # Extra help for OSX
  if [[ "$(uname -s)" == "Darwin" ]]; then
    echo
    echo "Ensure you are up to date on the following packages:"
    echo "$ brew install md5sha1sum bash"
  fi
  echo
fi

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
LICENSES_FILE="${REPO_ROOT}/LICENSES"
TMP_LICENSES_FILE="/tmp/cluster-registry.LICENSES.$$"
DEPS_DIR="vendor"

cd "${REPO_ROOT}"
declare -Ag CONTENT

# Put the cluster-registry LICENSE on top
(
echo "================================================================================"
echo "= cluster-registry licensed under: ="
echo
cat "${REPO_ROOT}/LICENSE"
echo
echo "= LICENSE $(md5sum "${REPO_ROOT}/LICENSE" | awk '{print $1}')"
echo "================================================================================"
) > "${TMP_LICENSES_FILE}"

# Loop through every package in vendor.
for PACKAGE in $(dep status | \
                 tail -n +2 |
                 awk '{print $1}' |
                 sort -f); do
  process_content ${PACKAGE} LICENSE
  process_content ${PACKAGE} COPYRIGHT

  # display content
  echo
  echo "================================================================================"
  echo "= ${DEPS_DIR}/${PACKAGE} licensed under: ="
  echo

  file=""
  if [[ -n "${CONTENT[${PACKAGE}-LICENSE]-}" ]]; then
      file="${CONTENT[${PACKAGE}-LICENSE]-}"
  elif [[ -n "${CONTENT[${PACKAGE}-COPYRIGHT]-}" ]]; then
      file="${CONTENT[${PACKAGE}-COPYRIGHT]-}"
  fi
  if [[ -z "${file}" ]]; then
      cat > /dev/stderr << __EOF__
No license could be found for ${PACKAGE} - aborting.

Options:
1. Check if the upstream repository has a newer version with LICENSE and/or
   COPYRIGHT files.
2. Contact the author of the package to ensure there is a LICENSE and/or
   COPYRIGHT file present.
3. Do not use this package in the cluster-registry.
__EOF__
      exit 9
  fi
  cat "${file}"

  echo
  echo "= ${file} $(md5sum ${file} | awk '{print $1}')"
  echo "================================================================================"
  echo
done >> "${TMP_LICENSES_FILE}"

if [[ ${1:-} == "--verify-only" ]]; then
  if ! _out="$(diff -Naupr "${LICENSES_FILE}" "${TMP_LICENSES_FILE}")"; then
    echo "Your LICENSES file is out of date. Run hack/update-licenses.sh and commit the results."
    echo "${_out}"
    exit 1
  fi
  exit 0
fi

mv -f "${TMP_LICENSES_FILE}" "${LICENSES_FILE}"
