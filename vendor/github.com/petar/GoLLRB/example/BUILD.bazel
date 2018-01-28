load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["ex1.go"],
    importpath = "github.com/petar/GoLLRB/example",
    visibility = ["//visibility:private"],
    deps = ["//vendor/github.com/petar/GoLLRB/llrb:go_default_library"],
)

go_binary(
    name = "example",
    importpath = "github.com/petar/GoLLRB/example",
    library = ":go_default_library",
    visibility = ["//visibility:public"],
)
