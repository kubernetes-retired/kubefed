<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Releasing Federation v2](#releasing-federation-v2)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

### Releasing Federation v2

Creating a federation v2 release involves the following steps:

1. Locally create an annotated tag in the format `v[0-9]+.[0-9]+.[0-9]+`
   - `git tag -a <tag> -m "Creating release tag <tag>"`
   - An annotated tag is required to ensure that the `kubefed2` and
     `controller-manager` binaries are versioned correctly.
   - At the time of writing, it is not possible to create an
     annotated tag through the github web interface.
2. Push the tag to master
   - `git push origin <tag>` (this requires write access to the repo)
3. Build `kubefed2` for release
   1. `make kubefed2`
   2. `cd bin` (from repo root)
   3. `kubefed2 version` (check that the output is as expected)
   4. `tar cvzf kubefed2.tgz kubefed2`
   5. `sha256sum kubefed2.tgz > kubefed2.tgz.sha`
4. Package the helm chart for release
   1. Update the default image tag in values.yaml (Change the version to match the release)
   2. Update the chart version in Chart.yaml (Format should be `x.x.x`)
   3. `cd charts` (from repo root)
   4. `helm package federation-v2`
   5. `sha256sum federation-v2-<x.x.x>.tgz > federation-v2-<x.x.x>.tgz.sha`
   6. `helm repo index . --merge index.yaml --url=https://github.com/kubernetes-sigs/federation-v2/releases/download/v<x.x.x>/federation-v2-<x.x.x>.tgz` (Add the new version to the chart index)
   7. Ensure index.yaml contains the added release
   8. Propose a PR that updates index.yaml
4. Create github release
   - Copy text from old release and replace old tag references
   - Add `kubefed2.tgz` and `kubefed2.tgz.sha`
   - Add `federation-v2-<x.x.x>.tgz` and `federation-v2-<x.x.x>.tgz.sha`
