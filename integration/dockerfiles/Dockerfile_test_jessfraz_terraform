FROM golang:alpine as builder
MAINTAINER Jessica Frazelle <jess@linux.com>

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

RUN	apk add --no-cache \
	bash \
	ca-certificates \
	gcc \
	git \
	make \
	musl-dev \
	zip

ENV TERRAFORM_VERSION v0.12.17

RUN go get github.com/hashicorp/terraform || true

WORKDIR /go/src/github.com/hashicorp/terraform

RUN git checkout "${TERRAFORM_VERSION}"

RUN XC_ARCH="amd64" XC_OS="linux" LD_FLAGS=" -extldflags -static" make fmt bin \
	&& mv bin/terraform /usr/bin/terraform

FROM alpine:latest

RUN apk add --no-cache \
	bash \
	tar

COPY --from=builder /usr/bin/terraform /usr/bin/terraform
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs

ENTRYPOINT [ "terraform" ]
CMD [ "--help" ]
