FROM alpine:latest

RUN apk --no-cache add \
	ca-certificates \
	perl-datetime \
	perl-timedate

ENV HOME /home/user
RUN adduser -u 1001 -D user \
	&& mkdir -p $HOME/.irssi \
	&& chown -R user:user $HOME

ENV LANG C.UTF-8

ENV IRSSI_VERSION 1.2.2
# https://otr.cypherpunks.ca/index.php#downloads
ENV LIB_OTR_VERSION 4.1.1
# https://github.com/cryptodotis/irssi-otr/releases
ENV IRSSI_OTR_VERSION 1.0.2

RUN set -x \
	&& apk add --no-cache --virtual .build-deps \
		autoconf \
		automake \
		curl \
		gcc \
		glib-dev \
		gnupg \
		libc-dev \
		libgcrypt-dev \
		libtool \
		lynx \
		make \
		ncurses-dev \
		openssl-dev \
		perl-dev \
		pkgconf \
		tar \
		xz \
	&& curl -sSL "https://github.com/irssi/irssi/releases/download/${IRSSI_VERSION}/irssi-${IRSSI_VERSION}.tar.xz" -o /tmp/irssi.tar.xz \
	&& curl -sSL "https://github.com/irssi/irssi/releases/download/${IRSSI_VERSION}/irssi-${IRSSI_VERSION}.tar.xz.asc" -o /tmp/irssi.tar.xz.asc \
	&& export GNUPGHOME="$(mktemp -d)" \
# gpg: key DDBEF0E1: public key "The Irssi project <staff@irssi.org>" imported
	&& gpg --no-tty --keyserver ha.pool.sks-keyservers.net --recv-keys 7EE65E3082A5FB06AC7C368D00CCB587DDBEF0E1 \
	&& gpg --batch --verify /tmp/irssi.tar.xz.asc /tmp/irssi.tar.xz \
	&& rm -rf "$GNUPGHOME" /tmp/irssi.tar.xz.asc \
	&& mkdir -p /usr/src \
	&& tar -xJf /tmp/irssi.tar.xz -C /usr/src \
	&& rm /tmp/irssi.tar.xz \
	&& ( \
		cd /usr/src/irssi-$IRSSI_VERSION \
		&& ./configure \
			--enable-true-color \
			--with-bot \
			--with-proxy \
			--with-socks \
			--prefix=/usr \
		&& make -j$(getconf _NPROCESSORS_ONLN) \
		&& make install \
	) \
	&& curl -sSL "https://otr.cypherpunks.ca/libotr-${LIB_OTR_VERSION}.tar.gz" -o /tmp/libotr.tar.gz \
	&& curl -sSL "https://otr.cypherpunks.ca/libotr-${LIB_OTR_VERSION}.tar.gz.asc" -o /tmp/libotr.tar.gz.asc \
	&& export GNUPGHOME="$(mktemp -d)" \
# gpg: key 42C2ABAD: public key "OTR Dev Team (Signing Key) <otr@cypherpunks.ca>" imported
	&& curl -sSL https://otr.cypherpunks.ca/gpgkey.asc | gpg --no-tty --import \
	&& gpg --batch --verify /tmp/libotr.tar.gz.asc /tmp/libotr.tar.gz \
	&& rm -rf "$GNUPGHOME" /tmp/libotr.tar.gz.asc \
	&& mkdir -p /usr/src/libotr \
	&& tar -xzf /tmp/libotr.tar.gz -C /usr/src/libotr --strip-components 1 \
	&& rm /tmp/libotr.tar.gz \
	&& ( \
		cd /usr/src/libotr \
		&& ./configure \
			--with-pic \
			--prefix=/usr \
		&& make \
		&& make install \
	) \
	&& mkdir -p /usr/src/irssi-otr \
	&& curl -sSL "https://github.com/cryptodotis/irssi-otr/archive/v${IRSSI_OTR_VERSION}.tar.gz" -o /tmp/irssi-otr.tar.gz \
	&& mkdir -p /usr/src/irssi-otr \
	&& tar -xf /tmp/irssi-otr.tar.gz -C /usr/src/irssi-otr --strip-components 1 \
	&& rm -f /tmp/irssi-otr.tar.gz \
	&& ( \
		cd /usr/src/irssi-otr \
		&& ./bootstrap \
		&& ./configure \
			--prefix=/usr \
		&& make \
		&& make install \
	) \
	&& rm -rf /usr/src/irssi-$IRSSI_VERSION \
	&& rm -rf /usr/src/libotr \
	&& rm -rf /usr/src/irssi-otr \
	&& runDeps="$( \
		scanelf --needed --nobanner --recursive /usr \
			| awk '{ gsub(/,/, "\nso:", $2); print "so:" $2 }' \
			| sort -u \
			| xargs -r apk info --installed \
			| sort -u \
	)" \
	&& apk add --no-cache --virtual .irssi-rundeps $runDeps perl-libwww \
	&& apk del .build-deps

WORKDIR $HOME
VOLUME $HOME/.irssi

USER user
CMD ["irssi"]
