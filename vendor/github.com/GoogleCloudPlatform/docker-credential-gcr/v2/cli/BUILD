load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "clear.go",
        "common.go",
        "config.go",
        "configure-docker.go",
        "dockerHelper.go",
        "gcr-login.go",
        "gcr-logout.go",
        "version.go",
    ],
    importpath = "github.com/GoogleCloudPlatform/docker-credential-gcr/v2/cli",
    visibility = ["//visibility:public"],
    deps = [
        "//auth:go_default_library",
        "//config:go_default_library",
        "//credhelper:go_default_library",
        "//store:go_default_library",
        "//vendor/github.com/docker/cli/cli/config:go_default_library",
        "//vendor/github.com/docker/cli/cli/config/configfile:go_default_library",
        "//vendor/github.com/docker/docker-credential-helpers/credentials:go_default_library",
        "//vendor/github.com/google/subcommands:go_default_library",
    ],
)
