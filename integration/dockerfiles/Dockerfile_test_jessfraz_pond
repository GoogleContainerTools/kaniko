FROM golang:latest as builder
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

RUN	apt-get update && apt-get install -y \
	libgtk-3-dev \
	libgtkspell3-3-dev \
	libtspi-dev \
	pkg-config \
	--no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

RUN go get -d -v github.com/agl/pond/client \
	&& go get -d -v github.com/agl/pond/server

WORKDIR /go/src/github.com/agl/pond

RUN go build -o /usr/bin/pond-client ./client \
	&& go build -o /usr/bin/pond-server ./server

FROM debian:bullseye-slim

RUN	apt-get update && apt-get install -y \
	ca-certificates \
	libgtk-3-0 \
	libgtkspell3-3-0 \
	libtspi1 \
	--no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

# create pond user
ENV HOME /home/pond
RUN useradd --create-home --home-dir $HOME pond \
    && chown -R pond:pond $HOME

WORKDIR $HOME
USER pond

COPY --from=builder /usr/bin/pond-client /usr/bin/pond-client
COPY --from=builder /usr/bin/pond-server /usr/bin/pond-server

CMD [ "pond-client", "-cli" ]
