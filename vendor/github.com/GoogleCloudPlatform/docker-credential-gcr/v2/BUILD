load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")
load("@bazel_gazelle//:def.bzl", "gazelle")

licenses(["notice"])  # Apache 2.0

exports_files(["LICENSE"])

gazelle(
    name = "gazelle",
    command = "fix",
    external = "vendored",
    extra_args = [
        "-build_file_name",
        "BUILD,BUILD.bazel",  # Prioritize `BUILD` for newly added files.
    ],
    prefix = "github.com/GoogleCloudPlatform/docker-credential-gcr",
)

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "github.com/GoogleCloudPlatform/docker-credential-gcr/v2",
    visibility = ["//visibility:private"],
    deps = [
        "//cli:go_default_library",
        "//vendor/github.com/google/subcommands:go_default_library",
    ],
)

go_binary(
    name = "docker-credential-gcr",
    embed = [":go_default_library"],
    pure = "on",
    visibility = ["//visibility:public"],
)
