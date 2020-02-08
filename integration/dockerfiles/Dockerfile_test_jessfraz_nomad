FROM golang:1.11 as builder
MAINTAINER Jessica Frazelle <jess@linux.com>

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

RUN apt-get update && apt-get install -y \
	apt-transport-https \
	ca-certificates \
	curl \
    --no-install-recommends \
    && rm -rf /var/lib/apt/lists/* \
	&& curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add - \
	&& echo "deb https://dl.yarnpkg.com/debian/ stable main" | tee /etc/apt/sources.list.d/yarn.list

RUN curl -sL https://deb.nodesource.com/setup_8.x | bash -

RUN apt-get update && apt-get install -y \
	gcc \
	git \
	g++ \
	make \
	nodejs \
	npm \
	pkgconf \
	python \
	yarn \
	zip \
    --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

ENV NOMAD_VERSION v0.10.2

RUN go get github.com/hashicorp/nomad

WORKDIR /go/src/github.com/hashicorp/nomad

RUN git checkout "${NOMAD_VERSION}"

RUN set -x \
	&& make bootstrap ember-dist static-assets \
	&& CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
		go build \
		-ldflags "-X github.com/hashicorp/nomad/version.GitCommit=$(git rev-parse HEAD) -extldflags -static " \
		-tags "ui release" \
		-o /usr/bin/nomad

FROM alpine:latest

COPY --from=builder /usr/bin/nomad /usr/bin/nomad
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs

ENTRYPOINT [ "nomad" ]
CMD [ "--help" ]
