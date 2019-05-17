<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Releasing KubeFed](#releasing-kubefed)

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
3. Build the release artifacts (will output to root of repo)
   - `./scripts/build-release-artifacts.sh <tag>`
4. Create github release
   1. Copy text from old release and replace old tag references
   2. Add a synopsis of the `Unreleased` section of `CHANGELOG.md`
   3. Add `kubefedctl-<x.x.x>-linux-amd64.tgz` and `kubefedctl-<x.x.x>-linux-amd64.tgz.sha`
   4. Add `kubefedctl-<x.x.x>-darwin-amd64.tgz` and `kubefedctl-<x.x.x>-darwin-amd64.tgz.sha`
   5. Add `kubefed-<x.x.x>.tgz` and `kubefed-<x.x.x>.tgz.sha`
5. Update master
   1. Move the contents of the `Unreleased` section of `CHANGELOG.md` to `v<x.x.x>`
   2. Propose a PR that includes changes to `charts/index.yaml`
      (updated by the build script) and `CHANGELOG.md`
