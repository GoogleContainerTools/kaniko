FROM golang:alpine as builder
LABEL maintainer "Christian Koep <christiankoep@gmail.com>"

RUN apk --no-cache add \
	ca-certificates \
	git \
	make

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

ENV MICRO_VERSION v1.4.1

RUN git clone --depth 1 --branch "$MICRO_VERSION" https://github.com/zyedidia/micro /go/src/github.com/zyedidia/micro

WORKDIR /go/src/github.com/zyedidia/micro

RUN make install

FROM alpine:latest

COPY --from=builder /go/bin/micro /usr/bin/micro
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs

ENTRYPOINT [ "micro" ]
