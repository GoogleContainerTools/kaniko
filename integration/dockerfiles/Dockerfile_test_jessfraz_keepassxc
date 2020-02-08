# keepassxc
#
# docker run -d \
#		-v /tmp/.X11-unix:/tmp/.X11-unix \
#		-v /etc/machine-id:/etc/machine-id:ro \
#		-v /usr/share/X11/xkb:/usr/share/X11/xkb/:ro \
#		-v $HOME/.config/keepassxc:/root/.config/keepassxc \
#		-e DISPLAY=unix$DISPLAY \
#		jess/keepassxc
#
FROM alpine:latest
LABEL maintainer "Christian Koep <christiankoep@gmail.com>"

ENV KEEPASSXC_VERSION 2.5.1

RUN buildDeps=' \
		automake \
		argon2-dev \
		bash \
		cmake \
		curl-dev \
		expat \
		g++ \
		gcc \
		git \
		libgcrypt-dev \
		libmicrohttpd-dev \
		libqrencode-dev	\
		libsodium-dev \
		make \
		qt5-qtbase-dev \
		qt5-qtsvg-dev \
		qt5-qttools-dev \
	' \
	set -x \
	&& apk --no-cache add \
		$buildDeps \
	&& git clone --depth 1 --branch ${KEEPASSXC_VERSION} https://github.com/keepassxreboot/keepassxc.git /usr/src/keepassxc \
	&& cd /usr/src/keepassxc \
	&& mkdir build \
	&& cd build \
	&& cmake -DWITH_TESTS=ON -DWITH_XC_AUTOTYPE=ON -DWITH_XC_HTTP=ON .. \
	&& make \
	&& make install \
	&& apk del $buildDeps \
	&& rm -rf /usr/src/keepassxc \
	&& echo "Build complete."

RUN	apk --no-cache add \
		argon2-libs \
		libcurl \
		libmicrohttpd \
		libgcrypt \
		libqrencode \
		libsodium \
		mesa-dri-intel \
		qt5-qtbase \
		qt5-qtbase-x11 \
		qt5-qtsvg \
		qt5-qttools \
		ttf-dejavu

ENTRYPOINT [ "/usr/local/bin/keepassxc" ]
