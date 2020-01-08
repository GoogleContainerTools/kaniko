FROM golang:1.10-alpine AS vndr
RUN  apk add --no-cache git
# NOTE: hack scripts override VNDR_VERSION to a specific revision
ARG VNDR_VERSION=master
RUN go get -d github.com/LK4D4/vndr \
  && cd /go/src/github.com/LK4D4/vndr \
	&& git checkout $VNDR_VERSION \
	&& go install ./
WORKDIR /go/src/github.com/moby/buildkit
COPY . .
# Remove vendor first to workaround  https://github.com/LK4D4/vndr/issues/63.
RUN rm -rf vendor
RUN vndr --verbose --strict
