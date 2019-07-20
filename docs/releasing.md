<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Releasing KubeFed](#releasing-kubefed)
- [How Image is Automatically Published](#how-image-is-automatically-published)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

### Releasing KubeFed

Creating a KubeFed release involves the following steps:

1. Locally create an annotated tag in the format `v[0-9]+.[0-9]+.[0-9]+`
   - `git tag -a <tag> -m "Creating release tag <tag>"`
   - An annotated tag is required to ensure that the `kubefedctl` and
     `controller-manager` binaries are versioned correctly.
   - At the time of writing, it is not possible to create an
     annotated tag through the github web interface.
2. Push the tag to master
   - `git push origin <tag>` (this requires write access to the repo)
3. Verify image builds, tags and pushes successfully
   1. Go to the [Travis branches](https://travis-ci.org/kubernetes-sigs/kubefed/branches)
      view to verify the build succeeded for the `<tag>` that was just pushed.
   2. If the latest build for the `<tag>` was successful, then navigate to the
      build view for the tag in question
      ([here](https://travis-ci.org/kubernetes-sigs/kubefed/builds/537843474)
      for example). At the bottom of the logs you should see the
      `after_success` section; which contains the logs for the step that
      handles building, tagging and pushing the built container image to the
      quay.io container registry. After expanding this section to view the
      detailed log and verifying the tagging and pushing was successful, you
      should note the sha256 digest for both the
      [`<tag>`](https://travis-ci.org/kubernetes-sigs/kubefed/builds/537843474#L3750)
      and the
      [`latest`](https://travis-ci.org/kubernetes-sigs/kubefed/builds/537843474#L3760)
      tags.
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

Our Travis CI system handles automatically building, tagging, and pushing the
`quay.io/kubernetes-multicluster/kubefed:<tag>` container image to our
repository whenever Travis detects that it's running on a commit from `master`
(uses `canary` image tag) or a git tag (uses `<tag>` and `latest` image tags).
This step is handled in the last phase of the Travis build within the
[`after_success` section of the Travis configuration
file](https://github.com/kubernetes-sigs/kubefed/blob/3e9223df55bfbf801bdd51da9a7ca79183fdff6d/.travis.yml#L28-L42).
