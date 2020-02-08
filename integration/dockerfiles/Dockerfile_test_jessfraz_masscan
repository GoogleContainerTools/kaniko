FROM alpine:latest
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

RUN apk add --no-cache \
	ca-certificates \
	libpcap-dev

ENV MASSCAN_VERSION 1.0.5

RUN set -x \
	&& apk add --no-cache --virtual .build-deps \
		build-base \
		clang \
		clang-dev \
		git \
		linux-headers \
	&& rm -rf /var/lib/apt/lists/* \
	&& git clone --depth 1 --branch "$MASSCAN_VERSION" https://github.com/robertdavidgraham/masscan.git /usr/src/masscan \
	&& ( \
	cd /usr/src/masscan \
	&& make \
	&& make install \
	) \
	&& rm -rf /usr/src/masscan \
	&& apk del .build-deps

ENTRYPOINT [ "masscan" ]
