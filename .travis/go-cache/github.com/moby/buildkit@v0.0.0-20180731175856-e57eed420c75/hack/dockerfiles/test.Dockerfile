ARG RUNC_VERSION=dd56ece8236d6d9e5bed4ea0c31fe53c7b873ff4
ARG CONTAINERD_VERSION=v1.1.0
# containerd v1.0 for integration tests
ARG CONTAINERD10_VERSION=v1.0.3
# available targets: buildkitd, buildkitd.oci_only, buildkitd.containerd_only
ARG BUILDKIT_TARGET=buildkitd
ARG REGISTRY_VERSION=2.6
ARG ROOTLESSKIT_VERSION=20b0fc24b305b031a61ef1a1ca456aadafaf5e77

# The `buildkitd` stage and the `buildctl` stage are placed here
# so that they can be built quickly with legacy DAG-unaware `docker build --target=...`

FROM golang:1.10-alpine AS gobuild-base
RUN apk add --no-cache g++ linux-headers
RUN apk add --no-cache git libseccomp-dev make

FROM gobuild-base AS buildkit-base
WORKDIR /go/src/github.com/moby/buildkit
COPY . .
RUN mkdir .tmp; \
  PKG=github.com/moby/buildkit VERSION=$(git describe --match 'v[0-9]*' --dirty='.m' --always) REVISION=$(git rev-parse HEAD)$(if ! git diff --no-ext-diff --quiet --exit-code; then echo .m; fi); \
  echo "-X ${PKG}/version.Version=${VERSION} -X ${PKG}/version.Revision=${REVISION} -X ${PKG}/version.Package=${PKG}" | tee .tmp/ldflags

FROM buildkit-base AS buildctl
ENV CGO_ENABLED=0
ARG GOOS=linux
RUN go build -ldflags "$(cat .tmp/ldflags) -d" -o /usr/bin/buildctl ./cmd/buildctl

FROM buildkit-base AS buildkitd
ENV CGO_ENABLED=1
RUN go build -installsuffix netgo -ldflags "$(cat .tmp/ldflags) -w -extldflags -static" -tags 'seccomp netgo cgo static_build' -o /usr/bin/buildkitd ./cmd/buildkitd

# test dependencies begin here
FROM gobuild-base AS runc
ARG RUNC_VERSION
ENV CGO_ENABLED=1
RUN git clone https://github.com/opencontainers/runc.git "$GOPATH/src/github.com/opencontainers/runc" \
	&& cd "$GOPATH/src/github.com/opencontainers/runc" \
	&& git checkout -q "$RUNC_VERSION" \
	&& go build -installsuffix netgo -ldflags '-w -extldflags -static' -tags 'seccomp netgo cgo static_build' -o /usr/bin/runc ./

FROM gobuild-base AS containerd-base
RUN apk add --no-cache btrfs-progs-dev
RUN git clone https://github.com/containerd/containerd.git /go/src/github.com/containerd/containerd
WORKDIR /go/src/github.com/containerd/containerd

FROM containerd-base as containerd
ARG CONTAINERD_VERSION
RUN git checkout -q "$CONTAINERD_VERSION" \
  && make bin/containerd \
  && make bin/containerd-shim \
  && make bin/ctr

# containerd v1.0 for integration tests
FROM containerd-base as containerd10
ARG CONTAINERD10_VERSION
RUN git checkout -q "$CONTAINERD10_VERSION" \
  && make bin/containerd \
  && make bin/containerd-shim

FROM buildkit-base AS unit-tests
COPY --from=runc /usr/bin/runc /usr/bin/runc
COPY --from=containerd /go/src/github.com/containerd/containerd/bin/containerd* /usr/bin/


FROM buildkit-base AS buildkitd.oci_only
ENV CGO_ENABLED=1
# mitigate https://github.com/moby/moby/pull/35456
WORKDIR /go/src/github.com/moby/buildkit
RUN go build -installsuffix netgo -ldflags "$(cat .tmp/ldflags) -w -extldflags -static" -tags 'no_containerd_worker seccomp netgo cgo static_build' -o /usr/bin/buildkitd.oci_only ./cmd/buildkitd

FROM buildkit-base AS buildkitd.containerd_only
ENV CGO_ENABLED=0
RUN go build -ldflags "$(cat .tmp/ldflags) -d"  -o /usr/bin/buildkitd.containerd_only -tags no_oci_worker ./cmd/buildkitd

FROM registry:$REGISTRY_VERSION AS registry

FROM gobuild-base AS rootlesskit-base
RUN git clone https://github.com/rootless-containers/rootlesskit.git /go/src/github.com/rootless-containers/rootlesskit
WORKDIR /go/src/github.com/rootless-containers/rootlesskit

FROM rootlesskit-base as rootlesskit
ARG ROOTLESSKIT_VERSION
# mitigate https://github.com/moby/moby/pull/35456
ENV GOOS=linux
RUN git checkout -q "$ROOTLESSKIT_VERSION" \
&& go build -o /rootlesskit ./cmd/rootlesskit

FROM unit-tests AS integration-tests
ENV BUILDKIT_INTEGRATION_ROOTLESS_IDPAIR="1000:1000"
RUN apk add --no-cache shadow shadow-uidmap sudo \
  && useradd --create-home --home-dir /home/user --uid 1000 -s /bin/sh user \
  && echo "XDG_RUNTIME_DIR=/run/user/1000; export XDG_RUNTIME_DIR" >> /home/user/.profile \
  && mkdir -m 0700 -p /run/user/1000 \
  && chown -R user /run/user/1000 /home/user
ENV BUILDKIT_INTEGRATION_CONTAINERD_EXTRA="containerd-1.0=/opt/containerd-1.0/bin"
COPY --from=containerd10 /go/src/github.com/containerd/containerd/bin/containerd* /opt/containerd-1.0/bin/
COPY --from=buildctl /usr/bin/buildctl /usr/bin/
COPY --from=buildkitd /usr/bin/buildkitd /usr/bin
COPY --from=registry /bin/registry /usr/bin
COPY --from=rootlesskit /rootlesskit /usr/bin/

FROM buildkit-base AS cross-windows
ENV GOOS=windows

FROM cross-windows AS buildctl.exe
RUN go build -ldflags "$(cat .tmp/ldflags)" -o /buildctl.exe ./cmd/buildctl

FROM cross-windows AS buildkitd.exe
ENV CGO_ENABLED=0
RUN go build -ldflags "$(cat .tmp/ldflags)" -o /buildkitd.exe ./cmd/buildkitd

FROM alpine AS buildkit-export
RUN apk add --no-cache git
VOLUME /var/lib/buildkit

# Copy together all binaries for oci+containerd mode
FROM buildkit-export AS buildkit-buildkitd
COPY --from=runc /usr/bin/runc /usr/bin/
COPY --from=buildkitd /usr/bin/buildkitd /usr/bin/
COPY --from=buildctl /usr/bin/buildctl /usr/bin/
ENTRYPOINT ["buildkitd"]

# Copy together all binaries needed for oci worker mode
FROM buildkit-export AS buildkit-buildkitd.oci_only
COPY --from=buildkitd.oci_only /usr/bin/buildkitd.oci_only /usr/bin/
COPY --from=buildctl /usr/bin/buildctl /usr/bin/
ENTRYPOINT ["buildkitd.oci_only"]

# Copy together all binaries for containerd worker mode
FROM buildkit-export AS buildkit-buildkitd.containerd_only
COPY --from=runc /usr/bin/runc /usr/bin/
COPY --from=buildkitd.containerd_only /usr/bin/buildkitd.containerd_only /usr/bin/
COPY --from=buildctl /usr/bin/buildctl /usr/bin/
ENTRYPOINT ["buildkitd.containerd_only"]

FROM alpine AS containerd-runtime
COPY --from=runc /usr/bin/runc /usr/bin/
COPY --from=containerd /go/src/github.com/containerd/containerd/bin/containerd* /usr/bin/
COPY --from=containerd /go/src/github.com/containerd/containerd/bin/ctr /usr/bin/
VOLUME /var/lib/containerd
VOLUME /run/containerd
ENTRYPOINT ["containerd"]

# Rootless mode.
# Still requires `--privileged`.
FROM buildkit-buildkitd AS rootless
RUN apk add --no-cache shadow shadow-uidmap \
  && useradd --create-home --home-dir /home/user --uid 1000 user \
  && mkdir -p /run/user/1000 /home/user/.local/tmp /home/user/.local/share/buildkit \
  && chown -R user /run/user/1000 /home/user
COPY --from=rootlesskit /rootlesskit /usr/bin/
USER user
ENV HOME /home/user
ENV USER user
ENV XDG_RUNTIME_DIR=/run/user/1000
ENV TMPDIR=/home/user/.local/tmp
VOLUME /home/user/.local/share/buildkit
ENTRYPOINT ["rootlesskit", "buildkitd"]

FROM buildkit-${BUILDKIT_TARGET}


