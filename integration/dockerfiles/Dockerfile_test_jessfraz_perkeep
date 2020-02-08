FROM golang:1.10-alpine AS builder
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

RUN	apk --no-cache add \
	ca-certificates \
	git

ENV PERKEEP_VERSION 0.10

RUN mkdir -p /go/src/perkeep.org \
	&& git clone --depth 1 --branch "${PERKEEP_VERSION}" https://camlistore.googlesource.com/camlistore.git /go/src/perkeep.org \
	&& cd /go/src/perkeep.org \
	&& go run make.go \
	&& cp -vr /go/bin/* /usr/local/bin/ \
	&& echo "Build complete."

FROM alpine:latest

RUN	apk --no-cache add \
	ca-certificates

COPY --from=builder /usr/local/bin/pk* /usr/bin/
COPY --from=builder /usr/local/bin/perkeepd /usr/bin/perkeepd

ENTRYPOINT [ "perkeepd" ]
