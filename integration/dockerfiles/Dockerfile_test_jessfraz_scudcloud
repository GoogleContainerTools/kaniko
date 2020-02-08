# To use:
# Needs X11 socket and dbus mounted
#
# docker run --rm -it \
#	-v /etc/machine-id:/etc/machine-id:ro \
#	-v /etc/localtime:/etc/localtime:ro \
#	-v /tmp/.X11-unix:/tmp/.X11-unix \
#	-e DISPLAY=unix$DISPLAY \
#	--device /dev/snd:/dev/snd \
#	-v /var/run/dbus:/var/run/dbus \
#	-v $HOME/.scudcloud:/home/user/.config/scudcloud \
#	--name scudcloud \
#	jess/scudcloud

FROM ubuntu:16.04
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && apt-get install -y \
	dbus-x11 \
	hunspell-en-us \
	libnotify-bin \
	python3-dbus \
	software-properties-common \
	--no-install-recommends && \
	apt-add-repository -y ppa:rael-gc/scudcloud && \
	apt-get update && \
	apt-get install -y \
	scudcloud \
	&& rm -rf /var/lib/apt/lists/*

ENV LANG en_US.UTF-8
ENV HOME /home/user
RUN useradd --create-home --home-dir $HOME user \
	&& chown -R user:user $HOME

USER user

ENTRYPOINT ["/usr/bin/scudcloud"]
