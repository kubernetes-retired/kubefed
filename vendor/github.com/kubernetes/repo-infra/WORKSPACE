workspace(name = "io_kubernetes_build")

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "5f3b0304cdf0c505ec9e5b3c4fc4a87b5ca21b13d8ecc780c97df3d1809b9ce6",
    url = "https://github.com/bazelbuild/rules_go/releases/download/0.15.1/rules_go-0.15.1.tar.gz",
)

load("@io_bazel_rules_go//go:def.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains()
