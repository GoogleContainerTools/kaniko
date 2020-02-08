FROM golang:alpine as builder
LABEL maintainer="Jessica Frazelle <jess@linux.com>"

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

RUN	apk add --no-cache \
	--repository http://dl-cdn.alpinelinux.org/alpine/edge/main/ \
	bash \
	ca-certificates \
	git \
	make \
	npm \
	yarn \
	zip

ENV VAULT_VERSION v1.3.0

RUN go get github.com/hashicorp/vault || true

WORKDIR /go/src/github.com/hashicorp/vault

RUN git checkout "${VAULT_VERSION}"

RUN XC_ARCH="amd64" XC_OS="linux" XC_OSARCH="linux/amd64" LD_FLAGS=" -extldflags -static " make bootstrap static-dist bin \
	&& mv bin/vault /usr/bin/vault

FROM alpine:latest

COPY --from=builder /usr/bin/vault /usr/bin/vault
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs

ENTRYPOINT [ "vault" ]
CMD [ "--help" ]
