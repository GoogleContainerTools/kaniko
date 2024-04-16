# Development

This doc explains the development workflow so you can get started
[contributing](CONTRIBUTING.md) to Kaniko!

## Getting started

First you will need to setup your GitHub account and create a fork:

1. Create [a GitHub account](https://github.com/join)
1. Setup [GitHub access via
   SSH](https://help.github.com/articles/connecting-to-github-with-ssh/)
1. [Create and checkout a repo fork](#checkout-your-fork)

Once you have those, you can iterate on kaniko:

1. [Run your instance of kaniko](README.md#running-kaniko)
1. [Verifying kaniko builds](#verifying-kaniko-builds)
1. [Run kaniko tests](#testing-kaniko)

When you're ready, you can [create a PR](#creating-a-pr)!

## Checkout your fork

The Go tools require that you clone the repository to the `src/github.com/GoogleContainerTools/kaniko` directory
in your [`GOPATH`](https://go.dev/wiki/SettingGOPATH).

To check out this repository:

1. Create your own [fork of this
  repo](https://help.github.com/articles/fork-a-repo/)
2. Clone it to your machine:

  ```shell
  mkdir -p ${GOPATH}/src/github.com/GoogleContainerTools
  cd ${GOPATH}/src/github.com/GoogleContainerTools
  git clone git@github.com:${YOUR_GITHUB_USERNAME}/kaniko.git
  cd kaniko
  git remote add upstream git@github.com:GoogleContainerTools/kaniko.git
  git remote set-url --push upstream no_push
  ```

_Adding the `upstream` remote sets you up nicely for regularly [syncing your
fork](https://help.github.com/articles/syncing-a-fork/)._

## Verifying kaniko builds

Images built with kaniko should be no different from images built elsewhere.
While you iterate on kaniko, you can verify images built with kaniko by:

1. Build the image using another system, such as `docker build`
2. Use [`container-diff`](https://github.com/GoogleContainerTools/container-diff) to diff the images

## Testing kaniko

kaniko has both [unit tests](#unit-tests) and [integration tests](#integration-tests).

Please note that the tests require a Linux machine - use Vagrant to quickly set
up the test environment needed if you work with macOS or Windows.

### Unit Tests

The unit tests live with the code they test and can be run with:

```shell
make test
```

_These tests will not run correctly unless you have [checked out your fork into your `$GOPATH`](#checkout-your-fork)._

### Lint Checks

The helper script to install and run lint is placed here at the root of project.

```shell
./hack/linter.sh
```

To fix any `gofmt` issues, you can simply run `gofmt` with `-w` flag like this

```shell
find . -name "*.go" | grep -v vendor/ | xargs gofmt -l -s -w
```

### Integration tests

Currently the integration tests that live in [`integration`](./integration) can be run against your own gcloud space or a local registry.

These tests will be kicked off by [reviewers](#reviews) for submitted PRs using GitHub Actions.

In either case, you will need the following tools:

* [`container-diff`](https://github.com/GoogleContainerTools/container-diff#installation)

#### GCloud

To run integration tests with your GCloud Storage, you will also need the following tools:

* [`gcloud`](https://cloud.google.com/sdk/install)
* [`gsutil`](https://cloud.google.com/storage/docs/gsutil_install)
* A bucket in [GCS](https://cloud.google.com/storage/) which you have write access to via
  the user currently logged into `gcloud`
* An image repo which you have write access to via the user currently logged into `gcloud`
* A docker account and a `~/.docker/config.json` with login credentials if you run
  into rate limiting problems during tests.

Once this step done, you must override the project using environment variables:

* `GCS_BUCKET` - The name of your GCS bucket
* `IMAGE_REPO` - The path to your docker image repo

This can be done as follows:

```shell
export GCS_BUCKET="gs://<your bucket>"
export IMAGE_REPO="gcr.io/somerepo"
```

Login for both user and application credentials
```shell
gcloud auth login
gcloud auth application-default login
```

Then you can launch integration tests as follows:

```shell
make integration-test
```

You can also run tests with `go test`, for example to run tests individually:

```shell
go test ./integration -v --bucket $GCS_BUCKET --repo $IMAGE_REPO -run TestLayers/test_layer_Dockerfile_test_copy_bucket
```

These tests will be kicked off by [reviewers](#reviews) for submitted PRs by the kokoro task.

#### Local integration tests

To run integration tests locally against a local registry and gcs bucket, set the LOCAL environment variable

```shell
LOCAL=1 make integration-test
```

#### Running integration tests for a specific dockerfile

In order to test only specific dockerfiles during local integration testing, you can specify a pattern to match against inside the integration/dockerfiles directory.

```shell
DOCKERFILE_PATTERN="Dockerfile_test_add*" make integration-test-run
```

This will only run dockerfiles that match the pattern `Dockerfile_test_add*`



### Benchmarking

The goal is for Kaniko to be at least as fast at building Dockerfiles as Docker is, and to that end, we've built
in benchmarking to check the speed of not only each full run, but also how long each step of each run takes. To turn
on benchmarking, just set the `BENCHMARK_FILE` environment variable, and kaniko will output all the benchmark info
of each run to that file location.

```shell
docker run -v $(pwd):/workspace -v ~/.config:/root/.config \
-e BENCHMARK_FILE=/workspace/benchmark_file \
gcr.io/kaniko-project/executor:latest \
--dockerfile=<path to Dockerfile> --context=/workspace \
--destination=gcr.io/my-repo/my-image
```
Additionally, the integration tests can output benchmarking information to a `benchmarks` directory under the
`integration` directory if the `BENCHMARK` environment variable is set to `true.`

```shell
BENCHMARK=true go test -v --bucket $GCS_BUCKET --repo $IMAGE_REPO
```

#### Benchmarking your GCB runs
If you are GCB builds are slow, you can check which phases in kaniko are bottlenecks or taking more time.
To do this, add "BENCHMARK_ENV" to your cloudbuild.yaml like this.
```shell script
steps:
- name: 'gcr.io/kaniko-project/executor:latest'
  args:
  - --build-arg=NUM=${_COUNT}
  - --no-push
  - --snapshot-mode=redo
  env:
  - 'BENCHMARK_FILE=gs://$PROJECT_ID/gcb/benchmark_file'
```
You can download the file `gs://$PROJECT_ID/gcb/benchmark_file` using `gsutil cp` command.

## Creating a PR

When you have changes you would like to propose to kaniko, you will need to:

1. Ensure the commit message(s) describe what issue you are fixing and how you are fixing it
   (include references to [issue numbers](https://help.github.com/articles/closing-issues-using-keywords/)
   if appropriate)
1. [Create a pull request](https://help.github.com/articles/creating-a-pull-request-from-a-fork/)

### Reviews

Each PR must be reviewed by a maintainer. This maintainer will add the `kokoro:run` label
to a PR to kick of [the integration tests](#integration-tests), which must pass for the PR
to be submitted.
