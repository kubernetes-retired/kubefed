<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Releasing Federation v2](#releasing-federation-v2)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

### Releasing Federation v2

Creating a federation v2 release involves the following steps:

1. Locally create an annotated tag in the format `v[0-9]+.[0-9]+.[0-9]+`
   - `git tag -a <tag> -m "Creating release tag <tag>"`
   - An annotated tag is required to ensure that the `kubefedctl` and
     `controller-manager` binaries are versioned correctly.
   - At the time of writing, it is not possible to create an
     annotated tag through the github web interface.
2. Push the tag to master
   - `git push origin <tag>` (this requires write access to the repo)
3. Build `kubefedctl` for release
   1. `make kubefedctl`
   2. `cd bin` (from repo root)
   3. `kubefedctl version` (check that the output is as expected)
   4. `tar cvzf kubefedctl.tgz kubefedctl`
   5. `sha256sum kubefedctl.tgz > kubefedctl.tgz.sha`
4. Package the helm chart for release
   1. Update the default image tag in values.yaml (Change the version to match the release)
   2. Update the chart version in Chart.yaml (Format should be `x.x.x`)
   3. `cd charts` (from repo root)
   4. `helm package federation-v2`
   5. `sha256sum federation-v2-<x.x.x>.tgz > federation-v2-<x.x.x>.tgz.sha`
   6. `helm repo index . --merge index.yaml --url=https://github.com/kubernetes-sigs/federation-v2/releases/download/v<x.x.x>` (Add the new version to the chart index)
   7. Check that index.yaml contains the added release
4. Create github release
   1. Copy text from old release and replace old tag references
   2. Add a synopsis of the `Unreleased` section of `CHANGELOG.md`
   3. Add `kubefedctl.tgz` and `kubefedctl.tgz.sha`
   4. Add `federation-v2-<x.x.x>.tgz` and `federation-v2-<x.x.x>.tgz.sha`
5. Update master
   1. Move the contents of the `Unreleased` section of `CHANGELOG.md` to `v<x.x.x>`
   2. Propose a PR that includes changes to `charts/index.yaml` (from `4.6`) and `CHANGELOG.md`
