FROM python:2-alpine AS buildbase
LABEL maintainer "Jess Frazelle <jess@linux.com>"

RUN apk add --no-cache \
	bash \
	go \
	git \
	gcc \
	g++ \
	libc-dev \
	libgcc \
	make

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

ENV DOCKER_AUTH_VERSION 1.3.1

RUN git clone --depth 1 --branch ${DOCKER_AUTH_VERSION} https://github.com/cesanta/docker_auth /go/src/github.com/cesanta/docker_auth

WORKDIR /go/src/github.com/cesanta/docker_auth/auth_server

RUN pip install GitPython
RUN make deps generate
RUN go build -o /usr/bin/auth_server --ldflags=--s

FROM alpine:latest

RUN	apk --no-cache add \
	ca-certificates

COPY --from=buildbase /usr/bin/auth_server /usr/bin/auth_server

ENTRYPOINT [ "auth_server" ]
CMD [ "/config/auth_config.yml" ]
