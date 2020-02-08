FROM golang:alpine AS builder

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

RUN apk --no-cache add \
		gcc \
		git \
		libc-dev


ENV CFSSL_VERSION 1.4.1

RUN git clone --depth 1 --branch "$CFSSL_VERSION" https://github.com/cloudflare/cfssl.git /go/src/github.com/cloudflare/cfssl

RUN set -x && \
	go get github.com/cloudflare/cfssl_trust/... && \
	go get github.com/GeertJohan/go.rice/rice && \
	cd /go/src/github.com/cloudflare/cfssl && rice embed-go -i=./cli/serve && \
	mkdir bin && cd bin && \
	go build ../cmd/cfssl && \
	go build ../cmd/cfssljson && \
	go build ../cmd/mkbundle && \
	go build ../cmd/multirootca && \
	echo "Build complete."

FROM alpine:latest

COPY --from=builder /go/src/github.com/cloudflare/cfssl_trust /etc/cfssl
COPY --from=builder /go/src/github.com/cloudflare/cfssl/bin/ /usr/bin

WORKDIR /etc/cfssl

EXPOSE 8888

ENTRYPOINT ["cfssl"]
CMD ["--help"]
