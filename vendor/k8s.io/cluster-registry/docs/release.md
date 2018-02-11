# Creating a release

> This release process is subject to change as the cluster-registry evolves.

Please see the [development doc](development.md#release-and-build-versioning)
for some more information about the release tools.

## Release process

You will need to have permissions to create a release and push tags on the
cluster-registry repo, as well as permissions for the `crreleases` GCP project,
in order to run this release process. We are working on determining how to
limit the amount of special privilege necessary to do a release.

1. Create an annotated tag in the cluster-registry repo. Choose the latest
   commit (or another commit if you have a particular reason not to choose the
   latest commit) and a tag name with the scheme `vX.Y.Z`. e.g.,
   `git tag -a v0.0.1`. Enter a short commit message, perhaps just the version
   name.
1. Push your tag to the upstream repo. Make sure you push upstream, not to your
   own fork: `git push upstream --tags`.
1. Create a
   [new release](https://github.com/kubernetes/cluster-registry/releases/new)
   on the GitHub Releases page for the cluster registry. Name the release
   `vX.Y.Z`. Leave the body empty; it will be added later. Select the tag you
   created in the first step.
1. Check out the tag that you just created: `git checkout tags/vX.Y.Z`.
1. Run `hack/release.sh $(git describe) >/tmp/relnotes`. This will require
   permissions for the `crreleases` GCP project, which you may not have. We are
   working on automating this step so that it does not require anything to be
   done on a local machine.
1. Paste the contents of the `relnotes` file into the body of the release.
1. Run `hack/update-openapi-spec.sh` and check in the updated OpenAPI spec. This
   means that the checked-in OpenAPI spec at the release tag will be a version
   behind, but for now this is an acceptable wart.
1. Send an announcement to
   [kubernetes-sig-multicluster](https://groups.google.com/forum/#!forum/kubernetes-sig-multicluster).

## Notes

- The cluster-registry does not use branches for its releases. As it becomes
  necessary, we will evaluate branching strategies.
- There is no verification process for releases. Since each commit is currently
  checked by per-PR tests that run the full suite of tests we have, we expect
  all commits to be green and suitable for release.
