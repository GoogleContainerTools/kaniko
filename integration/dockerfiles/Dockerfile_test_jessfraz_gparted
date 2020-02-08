# DESCRIPTION:	  Create gparted container with its dependencies
# AUTHOR:		  Jessie Frazelle <jess@linux.com>
# COMMENTS:
#	This file describes how to build a gparted container with all
#	dependencies installed. It uses native X11 unix socket.
#	Tested on Debian Jessie
# USAGE:
#	# Download gparted Dockerfile
#	wget https://raw.githubusercontent.com/jessfraz/dockerfiles/master/gparted/Dockerfile
#
#	# Build gparted image
#	docker build -t gparted .
#
#	docker run -v /tmp/.X11-unix:/tmp/.X11-unix \
#		--device=/dev/sda:/dev/sda \
#		--device=/dev/mem:/dev/mem \
#		--cap-add SYS_RAWIO \
#		-e DISPLAY=unix$DISPLAY gparted
#

# Base docker image
FROM debian:bullseye-slim
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

# Install Gparted and its dependencies
RUN apt-get update && apt-get install -y \
	dosfstools \
	gparted \
	libcanberra-gtk-module \
	procps \
	--no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

# Autorun gparted
CMD ["/usr/sbin/gparted"]
