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
	zip

ENV PACKER_VERSION v1.4.4

RUN go get github.com/hashicorp/packer

WORKDIR /go/src/github.com/hashicorp/packer

RUN git checkout "${PACKER_VERSION}"

RUN XC_ARCH="amd64" XC_OS="linux" LD_FLAGS=" -extldflags -static" make releasebin \
	&& mv bin/packer /usr/bin/packer

FROM alpine:latest

RUN apk add --no-cache \
	bash \
	tar

COPY --from=builder /usr/bin/packer /usr/bin/packer
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs

ENTRYPOINT [ "packer" ]
CMD [ "--help" ]
