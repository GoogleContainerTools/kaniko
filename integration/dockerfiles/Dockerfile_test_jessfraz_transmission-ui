# DESCRIPTION:	  Create transmission container with its dependencies
# AUTHOR:		  Jessie Frazelle <jess@linux.com>
# COMMENTS:
#	This file describes how to build a transmission container with all
#	dependencies installed. It uses native X11 unix socket.
#	Tested on Debian Jessie
# USAGE:
#	# Download transmission-ui Dockerfile
#	wget https://raw.githubusercontent.com/jessfraz/dockerfiles/master/transmission-ui/Dockerfile
#
#	# Build transmission image
#	docker build -t jess/transmission-ui .
#
#	docker run -v /tmp/.X11-unix:/tmp/.X11-unix \
#		-v /home/jessie/Torrents:/Torrents \
#		-e DISPLAY=unix$DISPLAY jess/transmission-ui
#

# Base docker image
FROM debian:bullseye-slim
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

# Install transmission and its dependencies
RUN apt-get update && apt-get install -y \
	transmission-cli \
	transmission-common \
	transmission-daemon \
	transmission-gtk \
	--no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

# Autorun transmission
CMD ["/usr/bin/transmission-gtk"]
