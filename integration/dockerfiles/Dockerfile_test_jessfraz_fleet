FROM golang:alpine AS builder
MAINTAINER Jessica Frazelle <jess@linux.com>

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

RUN apk add --no-cache \
	ca-certificates \
	gcc \
	git \
	make \
	npm \
	yarn

ENV FLEET_VERSION 2.4.0

RUN git clone --depth 1 --branch "${FLEET_VERSION}" https://github.com/kolide/fleet.git /go/src/github.com/kolide/fleet

WORKDIR /go/src/github.com/kolide/fleet

ENV GO111MODULE on

RUN make deps generate

RUN CGO_ENABLED=0 go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o /usr/bin/fleet ./cmd/fleet
RUN CGO_ENABLED=0 go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o /usr/bin/fleetctl ./cmd/fleetctl

FROM r.j3ss.co/osquery AS osquery

FROM scratch

COPY --from=builder /usr/bin/fleet /usr/bin/fleet
COPY --from=builder /usr/bin/fleetctl /usr/bin/fleetctl
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs
COPY --from=osquery /usr/share/osquery/packs /packs

ENTRYPOINT [ "fleet" ]
CMD [ "--help" ]
