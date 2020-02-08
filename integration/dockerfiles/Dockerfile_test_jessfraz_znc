# Run znc in a container
#
# docker run --restart always -d \
#	--name znc \
#	-p 6697:6697 \
#	-v /volumes/znc:/home/user/.znc \
#	jess/znc
#
FROM alpine:latest
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

RUN apk add --no-cache \
	ca-certificates \
	glib \
	perl \
	perl-datetime \
	perl-libwww

ENV HOME /home/user
RUN adduser -u 1001 -D user \
	&& chown -R user:user $HOME

ENV LANG C.UTF-8

ENV ZNC_VERSION 1.7.5

RUN set -x \
	&& apk add --no-cache --virtual .build-deps \
		build-base \
		curl \
		libressl-dev \
		perl-dev \
		tar \
	&& curl -sSL "http://znc.in/releases/znc-${ZNC_VERSION}.tar.gz" -o /tmp/znc.tar.gz \
	&& mkdir -p /usr/src/znc \
	&& tar -xzf /tmp/znc.tar.gz -C /usr/src/znc --strip-components 1 \
	&& rm /tmp/znc.tar.gz* \
	&& ( \
		cd /usr/src/znc \
		&& ./configure \
		&& make -j8 \
		&& make install \
	) \
	&& rm -rf /usr/src/znc \
	&& runDeps="$( \
		scanelf --needed --nobanner --recursive /usr \
			| awk '{ gsub(/,/, "\nso:", $2); print "so:" $2 }' \
			| sort -u \
			| xargs -r apk info --installed \
			| sort -u \
	)" \
	&& apk add --no-cache --virtual .irssi-rundeps $runDeps \
	&& apk del .build-deps

WORKDIR $HOME
USER user

ENTRYPOINT [ "znc" ]
CMD [ "-f" ]
