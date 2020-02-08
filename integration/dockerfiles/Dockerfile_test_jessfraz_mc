FROM alpine:latest
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

RUN apk add --no-cache \
		glib \
		libintl \
		libssh2 \
		ncurses-libs

ENV TERM xterm

# Create user and change ownership
RUN addgroup -g 1001 -S mc \
    && adduser -u 1001 -SHG mc mc \
    && mkdir -p /home/mc/.mc

ENV MC_VERSION 4.8.21

RUN set -x \
	&& apk add --no-cache --virtual .build-deps \
		aspell-dev \
		autoconf \
		automake \
		build-base \
		ca-certificates \
		curl \
		e2fsprogs-dev \
		gettext-dev \
		git \
		glib-dev \
		libssh2-dev \
		libtool \
		ncurses-dev \
		pcre-dev \
	&& git clone --depth 1 --branch "$MC_VERSION" https://github.com/MidnightCommander/mc.git /usr/src/mc \
	&& ( \
		cd /usr/src/mc \
		&& ./autogen.sh \
		&& ./configure \
			--prefix=/usr \
			--libexecdir=/usr/lib \
			--mandir=/usr/share/man \
			--sysconfdir=/etc \
			--enable-background \
			--enable-charset \
			--enable-largefile \
			--enable-vfs-sftp \
			--with-internal-edit \
			--with-mmap \
			--with-screen=ncurses \
			--with-subshell \
			--without-gpm-mouse \
			--without-included-gettext \
			--without-x \
			--enable-aspell \
		&& make \
		&& make install \
	) \
	&& curl -sSL "https://raw.githubusercontent.com/nkulikov/mc-solarized-skin/master/solarized.ini" > /home/mc/.mc/solarized.ini \
	&& rm -rf /usr/src/mc \
	&& apk del .build-deps \
	&& chown -R mc:mc /home/mc

ENV HOME=/home/mc

ENV MC_SKIN=${HOME}/.mc/solarized.ini

WORKDIR ${HOME}

ENTRYPOINT [ "mc" ]
