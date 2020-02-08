# plex home theater
#
# docker run -d -v /tmp/.X11-unix:/tmp/.X11-unix \
# 	-e DISPLAY=unix$DISPLAY \
# 	--device /dev/snd:/dev/snd \
# 	--device /dev/dri:/dev/dri \
# 	jess/plex-home-theater
#
FROM ubuntu:16.04
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

RUN apt-get update && apt-get install -y \
	software-properties-common \
	--no-install-recommends && \
	add-apt-repository ppa:plexapp/plexht && \
	add-apt-repository ppa:pulse-eight/libcec && \
	apt-get update && \
	apt-get install -y \
	plexhometheater \
	&& rm -rf /var/lib/apt/lists/*

ENTRYPOINT	[ "/usr/bin/plexhometheater.sh" ]
