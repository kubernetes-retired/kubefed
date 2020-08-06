# gazelle:repository_macro repos.bzl%go_repositories
workspace(name = "io_k8s_repo_infra")

load("//:load.bzl", "repositories")

repositories()

load("//:repos.bzl", "configure", "repo_infra_go_repositories")

configure()

repo_infra_go_repositories()
