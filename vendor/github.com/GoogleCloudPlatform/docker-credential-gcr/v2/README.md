<a href="https://gcr.io"><img src="https://avatars2.githubusercontent.com/u/21046548?s=400&v=4" height="120"/></a>

# docker-credential-gcr [![Build Status](https://github.com/GoogleCloudPlatform/docker-credential-gcr/actions/workflows/test.yml/badge.svg)](https://travis-ci.org/GoogleCloudPlatform/docker-credential-gcr) [![Go Report Card](https://goreportcard.com/badge/GoogleCloudPlatform/docker-credential-gcr)](https://goreportcard.com/report/GoogleCloudPlatform/docker-credential-gcr)

## Introduction

`docker-credential-gcr` is [Google Container Registry](https://cloud.google.com/container-registry/)'s _standalone_, `gcloud` SDK-independent Docker credential helper. It allows for **v18.03+ Docker clients** to easily make authenticated requests to GCR's repositories (gcr.io, eu.gcr.io, etc.).

**Note:** `docker-credential-gcr` is primarily intended for users wishing to authenticate with GCR in the **absence of `gcloud`**, though they are [not mutually exclusive](#gcr-credentials). For normal development setups, users are encouraged to use [`gcloud auth configure-docker`](https://cloud.google.com/sdk/gcloud/reference/auth/configure-docker), instead.

The helper implements the [Docker Credential Store](https://docs.docker.com/engine/reference/commandline/login/#/credentials-store) API, but enables more advanced authentication schemes for GCR's users. In particular, it respects [Application Default Credentials](https://developers.google.com/identity/protocols/application-default-credentials) and is capable of generating credentials automatically (without an explicit login operation) when running in App Engine or Compute Engine.

For even more authentication options, see GCR's documentation on [advanced authentication methods](https://cloud.google.com/container-registry/docs/advanced-authentication).

## GCR Credentials

_By default_, the helper searches for GCR credentials in the following order:

1. In the helper's private credential store (i.e. those stored via `docker-credential-gcr gcr-login`)
1. In a JSON file whose path is specified by the GOOGLE_APPLICATION_CREDENTIALS environment variable.
1. In a JSON file in a location known to the helper:
	* On Windows, this is `%APPDATA%/gcloud/application_default_credentials.json`.
	* On other systems, `$HOME/.config/gcloud/application_default_credentials.json`.
1. On Google App Engine, it uses the `appengine.AccessToken` function.
1. On Google Compute Engine, Kubernetes Engine, and App Engine Managed VMs, it fetches the credentials of the _service account_ associated with the VM from the metadata server (if available).

Users may limit, re-order how the helper searches for GCR credentials using `docker-credential-gcr config --token-source`. Number 1 above is designated by `store` and 2-5 by `env` (which cannot be individually restricted or re-ordered). Multiple sources are separated by commas, and the default is `"store,  env"`.

While it is recommended to use [`gcloud auth configure-docker`](https://cloud.google.com/sdk/gcloud/reference/auth/configure-docker) in `gcloud`-based work flows, you may optionally configure `docker-credential-gcr` to use `gcloud` as a token source (see example below).

**Examples:**

To use _only_ the gcloud SDK's access token:
```shell
docker-credential-gcr config --token-source="gcloud"
```

To search the environment, followed by the private store:
```shell
docker-credential-gcr config --token-source="env, store"
```

To verify that credentials are being returned for a given registry, e.g. for `https://gcr.io`:

```shell
echo "https://gcr.io" | docker-credential-gcr get
```

## Other Credentials

As of the 2.0 release, `docker-credential-gcr` no longer supports generalized [`credsStore`](https://docs.docker.com/engine/reference/commandline/login/#/credentials-store) functionality.

### Building from Source

The program in this repository is written with the Go programming language and can be built with `go build`. These instructions assume you are using [**Go 1.13+**](https://golang.org/) or higher.

You can download the source code, compile the binary, and put it in your `$GOPATH` with `go get`.

```shell
go get -u github.com/GoogleCloudPlatform/docker-credential-gcr/v2
```

If `$GOPATH/bin` is in your system `$PATH`, this will also automatically install the compiled binary. You can confirm using `which docker-credential-gcr` and continue to the [section on Configuration and Usage](#configuration-and-usage).

Alternatively, you can use `go build` to build the program. This creates a `docker-credential-gcr` executable.

```shell
cd $GOPATH/src/github.com/GoogleCloudPlatform/docker-credential-gcr
go build
```

Then, you can put that binary in your `$PATH` to make it visible to `docker`. For example, if `/usr/bin` is present in your system path:

```shell
sudo mv ./docker-credential-gcr /usr/bin/docker-credential-gcr
```

## Configuration and Usage

* Configure the Docker CLI to use `docker-credential-gcr` as a credential helper for the default set of GCR registries:

	```shell
	docker-credential-gcr configure-docker
	```

  To speed up `docker build`s, you can instead configure a minimal set of registries:

  ```shell
  docker-credential-gcr configure-docker --registries="eu.gcr.io, marketplace.gcr.io"
  ```

  * Alternatively, use the [manual configuration instructions](#manual-docker-client-configuration) below to configure your version of the Docker client.

* Log in to GCR (or don't! See the [GCR Credentials section](#gcr-credentials))

	```shell
	docker-credential-gcr gcr-login
	```

* Use Docker!

	```shell
	docker pull gcr.io/project-id/neato-container
	```

* Log out from GCR

	```shell
	docker-credential-gcr gcr-logout
	```

### Manual Docker Client Configuration

Add a `credHelpers` entry in the Docker config file (usually `~/.docker/config.json` on OSX and Linux, `%USERPROFILE%\.docker\config.json` on Windows) for each GCR registry that you care about. The key should be the domain of the registry (**without** the "https://") and the value should be the suffix of the credential helper binary (everything after "docker-credential-").

	e.g. for `docker-credential-gcr`:

  <pre>
    {
      "auths" : {
            ...
      },
      "credHelpers": {
            "coolregistry.com": ... ,
            <b>"gcr.io": "gcr",
            "asia.gcr.io": "gcr",
            ...</b>
      },
      "HttpHeaders": ...
      "psFormat": ...
      "imagesFormat": ...
      "detachKeys": ...
    }
  </pre>

## License

Apache 2.0. See [LICENSE](LICENSE) for more information.
