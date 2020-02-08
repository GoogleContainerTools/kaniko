# DESCRIPTION:	  Create transmission container with its dependencies
# AUTHOR:		  Jessie Frazelle <jess@linux.com>
# COMMENTS:
#	This file describes how to build a transmission container with all
#	dependencies installed.
#	Tested on Debian Jessie
# USAGE:
#	# Download transmission Dockerfile
#	wget https://raw.githubusercontent.com/jessfraz/dockerfiles/master/transmission/Dockerfile
#
#	# Build transmission image
#	docker build -t jess/transmission .
#
#	docker run -d --name transmission \
#		-v /home/jessie/Torrents:/transmission/download \
#		-p 9091:9091 -p 51413:51413 -p 51413:51413/udp \
#		jess/transmission
#

# Base docker image
FROM alpine:latest
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

# machine parsable metadata, for https://github.com/pycampers/dockapt
LABEL "registry_image"="r.j3ss.co/transmission"
LABEL "docker_run_flags"="-d --name transmission \
		-v ~/Downloads:/transmission/download \
		-p 9091:9091 -p 51413:51413 -p 51413:51413/udp"

RUN apk --no-cache add \
	transmission-daemon \
	&& mkdir -p /transmission/download \
		/transmission/watch \
		/transmission/incomplete \
		/transmission/config \
	&& chmod 1777 /transmission

ENV TRANSMISSION_HOME /transmission/config

EXPOSE 9091

ENTRYPOINT ["/usr/bin/transmission-daemon"]
CMD [ "--allowed", "127.*,10.*,192.168.*,172.16.*,172.17.*,172.18.*,172.19.*,172.20.*,172.21.*,172.22.*,172.23.*,172.24.*,172.25.*,172.26.*,172.27.*,172.28.*,172.29.*,172.30.*,172.31.*,169.254.*", "--watch-dir", "/transmission/watch", "--encryption-preferred", "--foreground", "--config-dir", "/transmission/config", "--incomplete-dir", "/transmission/incomplete", "--dht", "--no-auth", "--download-dir", "/transmission/download" ]
