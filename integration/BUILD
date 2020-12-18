load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "integration",
    srcs = [
        "cleanup.go",
        "cmd.go",
        "config.go",
        "gcs.go",
        "images.go",
    ],
    importpath = "github.com/GoogleContainerTools/kaniko/integration",
    tags = ["manual"],
    visibility = ["//visibility:public"],
    deps = ["//pkg/timing"],
)

go_test(
    name = "integration_test",
    srcs = [
        "benchmark_test.go",
        "integration_test.go",
        "integration_with_context_test.go",
        "integration_with_stdin_test.go",
        "k8s_test.go",
    ],
    data = glob(["testdata/**"]),
    embed = [":integration"],
    tags = ["manual"],
    deps = [
        "//pkg/timing",
        "//pkg/util",
        "//testutil",
        "//vendor/github.com/google/go-containerregistry/pkg/name",
        "//vendor/github.com/google/go-containerregistry/pkg/v1/daemon",
        "//vendor/github.com/pkg/errors",
    ],
)

load("@io_bazel_rules_docker//container:container.bzl", "container_image")
load("@io_bazel_rules_docker//contrib:test.bzl", "container_test")

ARCHITECTURES = [
    "amd64",
    "arm64",
]

# Image with testdata
[
    container_image(
        name = "buildtest_image_" + arch,
        architecture = arch,
        base = "//cmd/executor:image_" + arch,
        directory = "/workspace",
        files = [
            ":testdata/Dockerfile.trivial",
        ],
    )
    for arch in ARCHITECTURES
]

# Non-executable tests can run on any architecture,
# so do not tag them.
[
    container_test(
        name = "image_files_" + arch + "_test",
        configs = ["testdata/files.yaml"],
        image = "//cmd/executor:image_" + arch,
    )
    for arch in ARCHITECTURES
]

[
    container_test(
        name = "buildtest_image_" + arch + "_test",
        configs = [
            "testdata/files.yaml",
            "testdata/testfiles.yaml",
        ],
        image = ":buildtest_image_" + arch,
    )
    for arch in ARCHITECTURES
]

[
    container_test(
        name = "image_exec_" + arch + "_test",
        configs = ["testdata/exec.yaml"],
        image = "//cmd/executor:image_" + arch,
        tags = [
            "manual",
            arch,
        ],
    )
    for arch in ARCHITECTURES
]

[
    container_test(
        name = "image_build_" + arch + "_test",
        configs = ["testdata/build.yaml"],
        image = ":buildtest_image_" + arch,
        tags = [
            "manual",
            arch,
        ],
    )
    for arch in ARCHITECTURES
]
