# Sickbeard in a container
#
# docker run -d \
# 	--restart always \
#	-p 8081:8081 \
# 	-v /etc/localtime:/etc/localtime:ro \
# 	-v /volumes/sickbeard:/data \
#	--link transmission:transmission \
# 	--name sickbeard \
# 	jess/sickbeard
#
FROM alpine:latest
LABEL maintainer "Jessie Frazelle <jess@linux.com>"


RUN apk add --no-cache \
	--repository http://dl-cdn.alpinelinux.org/alpine/edge/community \
	ca-certificates \
	git \
	python \
	py-pip \
	py-setuptools

RUN pip install cheetah

ENV SICKBEARD_VERSION torrent_1080_subtitles

EXPOSE 8081

RUN git clone https://github.com/junalmeida/Sick-Beard.git /usr/src/sickbeard

WORKDIR /usr/src/sickbeard

ENTRYPOINT [ "python", "SickBeard.py" ]
CMD [ "--datadir", "/data" ]
