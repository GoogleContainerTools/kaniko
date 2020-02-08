FROM alpine:latest
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

RUN apk --no-cache add \
	bash \
	libffi \
	libressl \
	python3

ENV CERTBOT_VERSION 1.0.0

RUN buildDeps=' \
		build-base \
		git \
		libffi-dev \
		libressl-dev \
		python3-dev \
		py3-pip \
	' \
	set -x \
	&& apk --no-cache add $buildDeps \
	&& pip3 install acme \
	&& git clone --depth 1 --branch "v$CERTBOT_VERSION" https://github.com/certbot/certbot /usr/src/certbot \
	&& cd /usr/src/certbot/certbot \
	&& python3 setup.py build || return 1 \
	&& python3 setup.py install --prefix=/usr || return 1 \
	&& rm -rf /usr/src/certbot \
	&& apk del $buildDeps \
	&& echo "Build complete."

ENTRYPOINT [ "certbot" ]
