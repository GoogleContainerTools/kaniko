load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["login.go"],
    importpath = "github.com/GoogleCloudPlatform/docker-credential-gcr/v2/auth",
    visibility = ["//visibility:public"],
    deps = [
        "//config:go_default_library",
        "//vendor/github.com/toqueteos/webbrowser:go_default_library",
        "//vendor/golang.org/x/oauth2:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = ["login_integration_test.go"],
    embed = [":go_default_library"],
    deps = [
        "//config:go_default_library",
        "//vendor/golang.org/x/oauth2:go_default_library",
    ],
)
