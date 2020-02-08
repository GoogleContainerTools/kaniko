FROM golang:alpine as builder

RUN apk --no-cache add \
	ca-certificates \
	git \
	make

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

RUN go get github.com/quay/clair/cmd/clair || true

ENV CLAIR_VERSION v2.1.1

WORKDIR /go/src/github.com/quay/clair

RUN git checkout "${CLAIR_VERSION}"

RUN go install ./cmd/clair

FROM alpine:latest

RUN apk --no-cache add \
	ca-certificates \
	git \
	rpm \
	xz

COPY --from=builder /go/bin/clair /usr/bin/clair

ENTRYPOINT [ "clair" ]
