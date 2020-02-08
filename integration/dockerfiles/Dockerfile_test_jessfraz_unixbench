FROM alpine:latest
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

RUN apk add --no-cache \
	bash \
	coreutils \
	dateutils \
	gcc \
	make \
	musl-dev \
	perl

ENV UNIXBENCH_VERSION v5.1.3

RUN set -x \
	&& apk add --no-cache --virtual .build-deps \
		ca-certificates \
		curl \
	&& mkdir -p /usr/src/unixbench \
	&& curl -sSL "https://github.com/kdlucas/byte-unixbench/archive/${UNIXBENCH_VERSION}.tar.gz" | tar -xzC /usr/src/unixbench --strip-components 2  \
	&& chmod +x /usr/src/unixbench/Run \
	&& apk del .build-deps

WORKDIR /usr/src/unixbench

ENTRYPOINT [ "/usr/src/unixbench/Run" ]
