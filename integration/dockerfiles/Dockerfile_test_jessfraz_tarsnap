FROM alpine:latest
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

RUN apk add --no-cache \
	ca-certificates \
	gnupg \
	&& rm -rf /var/lib/apt/lists/*

ENV TARSNAP_VERSION 1.0.39

RUN set -x \
	&& apk add --no-cache --virtual .build-deps \
		build-base \
		curl \
		e2fsprogs-libs \
		e2fsprogs-dev \
		make \
		openssl-dev \
		perl-digest-sha1 \
		perl-utils \
		tar \
		zlib-dev \
	&& curl -sSL "https://www.tarsnap.com/download/tarsnap-autoconf-${TARSNAP_VERSION}.tgz" -o /tmp/tarsnap.tgz \
	&& curl -sSL "https://www.tarsnap.com/download/tarsnap-sigs-${TARSNAP_VERSION}.asc" -o /tmp/tarsnap.tgz.asc \
	&& curl -sSL "https://www.tarsnap.com/tarsnap-signing-key-2015.asc" | gpg --no-tty --import \
	&& sha=$(gpg --decrypt /tmp/tarsnap.tgz.asc | awk '{ print $4 }') \
	&& if [ "$sha" != "$(shasum -a 256 /tmp/tarsnap.tgz | awk '{ print $1 }')" ]; then exit 1; fi \
	&& mkdir -p /usr/src/tarsnap \
	&& tar -xzf /tmp/tarsnap.tgz -C /usr/src/tarsnap --strip-components 1 \
	&& rm /tmp/tarsnap.tgz* \
	&& ( \
		cd /usr/src/tarsnap \
		&& ./configure --prefix=/usr \
		&& make \
		&& make install \
	) \
	&& rm -rf /usr/src/tarsnap \
	&& apk del .build-deps

ENTRYPOINT [ "tarsnap" ]
CMD [ "--help" ]
