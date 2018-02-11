# These dependencies' versions are pulled from the k/k WORKSPACE.
# https://github.com/kubernetes/kubernetes/blob/77ac663df427d1ae0cb45adb0a3eba263809c837/build/root/WORKSPACE
http_archive(
    name = "io_bazel_rules_go",
    sha256 = "0efdc3cca8ac1c29e1c837bee260dab537dfd373eb4c43c7d50246a142a7c098",
    strip_prefix = "rules_go-74d8ad8f9f59a1d9a7cf066d0980f9e394acccd7",
    urls = ["https://github.com/bazelbuild/rules_go/archive/74d8ad8f9f59a1d9a7cf066d0980f9e394acccd7.tar.gz"],
)

http_archive(
    name = "io_kubernetes_build",
    sha256 = "f4946917d95c54aaa98d1092757256e491f8f48fd550179134f00f902bc0b4ce",
    strip_prefix = "repo-infra-c75960142a50de16ac6225b0843b1ff3476ab0b4",
    urls = ["https://github.com/kubernetes/repo-infra/archive/c75960142a50de16ac6225b0843b1ff3476ab0b4.tar.gz"],
)

load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains", "go_repository")

go_rules_dependencies()

go_register_toolchains(go_version = "1.9.2")

# Docker rules
git_repository(
    name = "io_bazel_rules_docker",
    remote = "https://github.com/bazelbuild/rules_docker.git",
    tag = "v0.3.0",
)

load("@io_bazel_rules_docker//docker:docker.bzl", "docker_repositories", "docker_pull")

docker_repositories()

docker_pull(
    name = "ubuntu",
    digest = "sha256:34471448724419596ca4e890496d375801de21b0e67b81a77fd6155ce001edad",
    registry = "index.docker.io",
    repository = "library/ubuntu",
)
