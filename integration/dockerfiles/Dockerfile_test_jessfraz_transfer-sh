FROM golang:alpine as builder
LABEL maintainer "Jess Frazelle <jess@linux.com>"

RUN	apk --no-cache add \
	ca-certificates \
	git

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

ENV TRANSFER_SH_VERSION master

RUN	git clone --depth 1 --branch ${TRANSFER_SH_VERSION} https://github.com/dutchcoders/transfer.sh /go/src/github.com/dutchcoders/transfer.sh

WORKDIR /go/src/github.com/dutchcoders/transfer.sh

RUN GO111MODULE=on go build -o /usr/bin/transfer.sh

# Create a clean image without build dependencies
FROM alpine:latest

COPY --from=builder /usr/bin/transfer.sh /usr/bin/transfer.sh
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs

ENTRYPOINT [ "transfer.sh" ]
CMD [ "--help" ]
