load("@io_bazel_rules_docker//container:container.bzl", "container_image")

container_image(
    name = "nsswitch",
    directory = "etc",
    files = [":nsswitch.conf"],
    visibility = ["//visibility:private"],
)

container_image(
    name = "image",
    base = ":nsswitch",
    directory = "kaniko/ssl/certs",
    files = [":ca-certificates.crt"],
    visibility = ["//visibility:public"],
)
