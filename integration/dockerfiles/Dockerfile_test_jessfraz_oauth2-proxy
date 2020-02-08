FROM golang:1.12-alpine as builder
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

RUN	apk --no-cache add \
	ca-certificates \
	git

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

ENV OAUTH2_PROXY_VERSION v4.0.0

RUN go get github.com/golang/dep/cmd/dep

RUN go get github.com/pusher/oauth2_proxy || true \
	&& cd /go/src/github.com/pusher/oauth2_proxy \
	&& git checkout "${OAUTH2_PROXY_VERSION}" \
	&& go build . \
	&& mv oauth2_proxy /usr/bin/


FROM alpine:latest

COPY --from=builder /usr/bin/oauth2_proxy /usr/bin/oauth2_proxy
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs

ENTRYPOINT [ "oauth2_proxy" ]
