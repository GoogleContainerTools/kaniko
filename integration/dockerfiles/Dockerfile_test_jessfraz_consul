FROM golang:1.12 as builder
MAINTAINER Jessica Frazelle <jess@linux.com>

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

RUN apt-get update && apt-get install -y \
	apt-transport-https \
	ca-certificates \
	curl \
	gcc \
	git \
	make \
	ruby-dev \
	ruby \
	--no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

# Add yarn to apt repos.
RUN curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add -
RUN echo "deb https://dl.yarnpkg.com/debian/ stable main" | tee /etc/apt/sources.list.d/yarn.list

# Setup node install.
RUN curl -sL https://deb.nodesource.com/setup_10.x | bash -

RUN apt-get update && apt-get install -y \
	nodejs \
	yarn \
	--no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

ENV CONSUL_VERSION v1.6.2

RUN go get github.com/hashicorp/consul || true

WORKDIR /go/src/github.com/hashicorp/consul

RUN git checkout "${CONSUL_VERSION}"

# Install deps for UI.
RUN gem install \
	bundler \
	--no-ri --no-rdoc

RUN cd ui-v2 \
	&& yarn install \
	&& make \
	&& rm -rf ../pkg/web_ui \
	&& mkdir -p ../pkg \
	&& cp -r dist ../pkg/web_ui

RUN TERM=xterm XC_ARCH="amd64" XC_OS="linux" LD_FLAGS=" -extldflags -static" make tools static-assets bin \
	&& mv bin/consul /usr/bin/consul

FROM alpine:latest

COPY --from=builder /usr/bin/consul /usr/bin/consul
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs

ENTRYPOINT [ "consul" ]
CMD [ "--help" ]
