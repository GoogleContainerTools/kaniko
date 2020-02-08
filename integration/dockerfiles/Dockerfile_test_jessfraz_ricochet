# Run ricochet in a container
# see: https://ricochet.im/
#
# docker run -d \
#	--restart always \
#	-v /etc/localtime:/etc/localtime:ro \
#	-v /tmp/.X11-unix:/tmp/.X11-unix \
#	-e DISPLAY=unix$DISPLAY \
# 	--name ricochet \
# 	jess/ricochet
#
FROM debian:sid-slim
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

ENV DEBIAN_FRONTEND noninteractive

RUN mkdir -p /etc/xdg/QtProject && \
	apt-get update && apt-get install -y \
	dirmngr \
	gnupg \
	libasound2 \
	libfontconfig1 \
	libgl1-mesa-dri \
	libgl1-mesa-glx \
	libx11-xcb1 \
	libxext6 \
	libxrender1 \
	qtbase5-dev \
	&& rm -rf /var/lib/apt/lists/*

ENV RICOCHET_VERSION 1.1.4
ENV RICOCHET_FINGERPRINT 0xFF97C53F183C045D

RUN buildDeps=' \
		bzip2 \
		ca-certificates \
		curl \
	' \
	&& set -x \
	&& apt-get update && apt-get install -y $buildDeps --no-install-recommends \
	&& rm -rf /var/lib/apt/lists/* \
	&& curl -sSL "https://ricochet.im/releases/${RICOCHET_VERSION}/ricochet-${RICOCHET_VERSION}-linux-x86_64.tar.bz2" -o /tmp/ricochet.tar.bz2 \
	&& curl -sSL "https://ricochet.im/releases/${RICOCHET_VERSION}/ricochet-${RICOCHET_VERSION}-linux-x86_64.tar.bz2.asc" -o /tmp/ricochet.tar.bz2.asc \
	&& export GNUPGHOME="$(mktemp -d)" \
	&& chmod 600 "${GNUPGHOME}" \
	&& curl -sSL https://ricochet.im/john-brooks.asc | gpg --no-tty --import \
	&& gpg --fingerprint --keyid-format LONG ${RICOCHET_FINGERPRINT} | grep "9032 CAE4 CBFA 933A 5A21  45D5 FF97 C53F 183C 045D" \
	&& gpg --batch --verify /tmp/ricochet.tar.bz2.asc /tmp/ricochet.tar.bz2 \
	&& tar -vxj --strip-components 1 -C /usr/local/bin -f /tmp/ricochet.tar.bz2 \
	&& rm -rf /tmp/ricochet* \
	&& rm -rf "${GNUPGHOME}" \
	&& apt-get purge -y --auto-remove $buildDeps

ENTRYPOINT [ "ricochet" ]
