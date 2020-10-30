load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")
load("@io_bazel_rules_docker//container:container.bzl", "container_image")

go_library(
    name = "executor_lib",
    srcs = ["main.go"],
    importpath = "github.com/GoogleContainerTools/kaniko/cmd/executor",
    visibility = ["//visibility:private"],
    deps = ["//cmd/executor/cmd"],
)

go_binary(
    name = "executor",
    embed = [":executor_lib"],
    pure = "on",
    visibility = ["//visibility:public"],
)

ARCHITECTURES = [
    "amd64",
    "arm64",
    "s390x",
]

[
    go_binary(
        name = "executor_" + arch,
        embed = [":executor_lib"],
        goarch = arch,
        goos = "linux",
        pure = "on",
        visibility = ["//visibility:public"],
    )
    for arch in ARCHITECTURES
]

[
    container_image(
        name = "image_" + arch,
        architecture = arch,
        base = "//files:image",
        directory = "/kaniko",
        entrypoint = ["/kaniko/executor_" + arch],
        env = {
            "HOME": "/root",
            "USER": "root",
            "PATH": "/usr/local/bin:/kaniko",
            "SSL_CERT_DIR": "/kaniko/ssl/certs",
            "DOCKER_CONFIG": "/kaniko/.docker/",
        },
        files = [
            ":executor_" + arch,
        ],
        symlinks = {
            "/kaniko/executor": "/kaniko/executor_" + arch,
        },
        visibility = ["//visibility:public"],
        workdir = "/workspace",
    )
    for arch in ARCHITECTURES
]
