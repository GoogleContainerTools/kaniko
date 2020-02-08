FROM golang:alpine as builder
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

RUN	apk --no-cache add \
	ca-certificates \
	git

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

RUN go get github.com/kellegous/go || true \
	&& cd /go/src/github.com/kellegous/go \
	&& go build ./cmd/go \
	&& mv go /usr/bin/go


FROM alpine:latest

COPY --from=builder /usr/bin/go /usr/bin/go
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs

ENTRYPOINT [ "go" ]
