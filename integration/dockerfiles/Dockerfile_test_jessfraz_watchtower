FROM golang:alpine as builder
LABEL maintainer "Jess Frazelle <jess@linux.com>"

RUN apk --no-cache add \
	ca-certificates \
	gcc \
	git \
	libc-dev

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

ENV WATCHTOWER_VERSION v0.3.9

RUN git clone --depth 1 --branch ${WATCHTOWER_VERSION} https://github.com/containrrr/watchtower /go/src/github.com/containrrr/watchtower

WORKDIR /go/src/github.com/containrrr/watchtower

RUN GO111MODULE=on go build -o /usr/bin/watchtower

FROM alpine:latest
COPY --from=builder /usr/bin/watchtower /usr/bin/watchtower
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs

ENTRYPOINT [ "watchtower" ]
CMD [ "--help" ]
