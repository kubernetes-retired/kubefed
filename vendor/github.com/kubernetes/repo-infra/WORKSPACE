workspace(name = "io_kubernetes_build")

git_repository(
    name = "io_bazel_rules_go",
    commit = "f676870c5caf8df559a51e7aa005d2ece148a03b",
    remote = "https://github.com/bazelbuild/rules_go.git",
)

load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains")

go_rules_dependencies()

go_register_toolchains()
