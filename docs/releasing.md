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
    2. `cd bin`
    3. `kubefed2 version` (check that the output is as expected)
    4. `tar cvzf kubefed2.tar.gz kubefed2`
    5. `sha256sum kubefed2.tar.gz > kubefed2.tar.gz.sha`
4. Create github release
    - Copy text from old release and replace old tag references
    - Add `kubefed2.tar.gz` and `kubefed2.tar.gz.sha`
