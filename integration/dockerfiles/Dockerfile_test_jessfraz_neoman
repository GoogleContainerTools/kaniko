# Run neoman (yubikey-piv-manager) in a container
#
# docker run -d \
#	-v /etc/localtime:/etc/localtime:ro \
#	-v /tmp/.X11-unix:/tmp/.X11-unix \
#	-e DISPLAY=unix$DISPLAY \
#	--device /dev/bus/usb \
#	--device /dev/usb \
#	--name neoman \
#	jess/neoman
#
FROM ubuntu:16.04
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

RUN apt-get update && apt-get install -y \
	software-properties-common \
	--no-install-recommends && \
	add-apt-repository ppa:yubico/stable && \
	apt-get update && \
	apt-get install -y \
	python-setuptools \
	usbutils \
	yubikey-neo-manager \
	&& rm -rf /var/lib/apt/lists/*

ENTRYPOINT	[ "neoman" ]
