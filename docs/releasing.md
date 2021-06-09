<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Releasing KubeFed](#releasing-kubefed)
  - [Automated](#automated)
      - [Prerequisites](#prerequisites)
      - [Create Release](#create-release)
  - [Manual](#manual)
  - [How Image is Automatically Published](#how-image-is-automatically-published)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Releasing KubeFed

### Automated

Creating a KubeFed release can be automated by running a couple of scripts.

##### Prerequisites

The scripts used to automated the release of KubeFed rely on the following
prereqs:

- [hub CLI](https://github.com/github/hub#installation)
    - Once installed verify you can run `hub release show -f "%n" v0.1.0-rc5`
      without prompting for GitHub authentication.
- [jq](https://stedolan.github.io/jq/)
- helm. See
  https://github.com/kubernetes-sigs/kubefed/blob/master/docs/development.md#binaries
  for details.
- git remote naming conventions used by GitHub's `hub` CLI tool:
    - `origin` GitHub remote name for KubeFed fork. This can be overriden with
      `GITHUB_REMOTE_FORK_NAME=<remote>`.
    - `upstream` GitHub remote name for upstream KubeFed repo. This can be
      overriden with `GITHUB_REMOTE_UPSTREAM_NAME=<remote>`.

##### Create Release

1. `./scripts/build-release.sh <RELEASE_TAG>`

    - This step builds the release artifacts, creates the annotated
      `RELEASE_TAG` and pushes it to master at `GITHUB_REMOTE_UPSTREAM_NAME`.
      It then verifies the Github Action starts and completes successfully
      followed by verification that the container image built is successfully
      pushed to Quay.

2. Edit the `CHANGELOG.md` for correct wording and leave it uncommitted. Then proceed to the next
   step.
3. `./scripts/create-gh-release.sh <RELEASE_TAG>`
    - This step creates a pull request with the release `RELEASE_TAG` changes
      and creates a GitHub draft release. It outputs the URLs for each at the
      end of the script execution for manual merging and publishing when ready.

### Manual

Creating a KubeFed release involves the following steps:

1. Locally create an annotated tag in the format `v[0-9]+.[0-9]+.[0-9]+(-(alpha|beta|rc)\.?[0-9]+)?`
   - `git tag -s -a <tag> -m "Creating release tag <tag>"`
   - An annotated tag is required to ensure that the `kubefedctl` and
     `controller-manager` binaries are versioned correctly.
   - At the time of writing, it is not possible to create an
     annotated tag through the github web interface.
2. Push the tag to master
   - `git push origin <tag>` (this requires write access to the repo)
3. Verify image builds, tags and pushes successfully
   1. Go to the [Github Actions](https://github.com/kubernetes-sigs/kubefed/actions)
      view to verify the build succeeded for the `<tag>` that was just pushed.
   2. If the latest build for the `<tag>` was successful, then navigate to the
      build view for the tag in question
      ([here](https://github.com/kubernetes-sigs/kubefed/runs/2782637960)
      for example). At the bottom of the logs you should see the
      step that handles building, tagging and pushing the built container image to
      the quay.io container registry. After expanding this section to view the
      detailed log and verifying the tagging and pushing was successful, you
      should note the sha256 digest for both the
      `<tag>` and the `latest` image tags.
   3. You can verify the `kubefed` image with these tags were successfully
      pushed to the [quay.io/kubernetes-multicluster
      repository](https://quay.io/repository/kubernetes-multicluster/kubefed)
      by using the sha256 digests obtained in the previous step and verifying
      it is the same reported in the [quay.io/kubernetes-multicluster
      repository tags
      view](https://quay.io/repository/kubernetes-multicluster/kubefed?tab=tags).
      You can also verify the `LAST MODIFIED` timestamp coincides with the time
      the images were built. Additionally, quay.io provides a "link" symbol
      next to the sha256 digest under the `MANIFEST` column to indicate that
      the image tags `latest` and `<tag>` are identical. Hovering over this
      "link" symbol shows the tags having the same common image.
4. Build the release artifacts (will output to root of repo).
   - Make sure `helm` is in your `PATH`. If not, execute:
     - `./scripts/download-binaries.sh`
     - `export PATH=$(pwd)/bin:${PATH}`
   - `./scripts/build-release-artifacts.sh <tag>`
5. Create github release
   1. Copy text from old release and replace old tag references
   2. Add a synopsis of the `Unreleased` section of `CHANGELOG.md`
   3. Add `kubefedctl-<x.x.x>-linux-amd64.tgz` and `kubefedctl-<x.x.x>-linux-amd64.tgz.sha`
   4. Add `kubefedctl-<x.x.x>-darwin-amd64.tgz` and `kubefedctl-<x.x.x>-darwin-amd64.tgz.sha`
   5. Add `kubefed-<x.x.x>.tgz` and `kubefed-<x.x.x>.tgz.sha`
6. Update master
   1. Move the contents of the `Unreleased` section of `CHANGELOG.md` to `v<x.x.x>`
   2. Propose a PR that includes changes to `charts/index.yaml`
      (updated by the build script) and `CHANGELOG.md`

### How Image is Automatically Published

Our Github Actions workflows handle automatic building, tagging, and pushing of the
`quay.io/kubernetes-multicluster/kubefed:<tag>` container image to our
repository whenever Github Actions detects that it's running on a commit from `master`
(uses `canary` image tag) or a git tag (uses `<tag>` and `latest` image tags).
