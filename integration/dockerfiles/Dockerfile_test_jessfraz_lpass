FROM alpine:latest
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

RUN apk add --no-cache \
	ca-certificates \
	curl \
	libxml2 \
	libressl \
	xclip \
	--repository http://dl-cdn.alpinelinux.org/alpine/edge/main

RUN set -x \
	&& apk add --no-cache --virtual .build-deps \
		autoconf \
		automake \
		build-base \
		cmake \
		curl-dev \
		git \
		libressl-dev \
		libxml2-dev \
	&& git clone --depth 1 https://github.com/LastPass/lastpass-cli.git /usr/src/lastpass-cli \
	&& ( \
		cd /usr/src/lastpass-cli \
		&& cmake . \
		&& make \
		&& make install \
	) \
	&& rm -rf /usr/src/lastpass-cli \
	&& apk del .build-deps

ENTRYPOINT [ "lpass" ]
