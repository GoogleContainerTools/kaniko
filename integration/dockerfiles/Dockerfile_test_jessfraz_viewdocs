FROM golang:alpine as builder
MAINTAINER Jessica Frazelle <jess@linux.com>

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

RUN	apk add --no-cache \
	ca-certificates \
	git

RUN go get github.com/progrium/viewdocs

WORKDIR /go/src/github.com/progrium/viewdocs

RUN CGO_ENABLED=0 go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o /usr/bin/viewdocs *.go

FROM scratch

COPY --from=builder /usr/bin/viewdocs /usr/bin/viewdocs
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs

ENTRYPOINT [ "viewdocs" ]
CMD [ "--help" ]
