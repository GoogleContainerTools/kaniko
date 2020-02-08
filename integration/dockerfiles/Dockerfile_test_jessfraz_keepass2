# DESCRIPTION:	Create keepass2 container with its dependencies
# AUTHOR:		Christian Koep <christiankoep@gmail.com>
# USAGE:
#	# Build keepass2 image
#	docker build -t keepass2 .
#
#	# Run the container and mount your keepass2 database file
#	docker run -it \
#		-v /home/$USER/DB.kdbx:/root/DB.kdbx \
#		-v /tmp/.X11-unix:/tmp/.X11-unix \
#		-v /home/$USER/keepass2-plugins:/usr/lib/keepass2/Plugins \
#		-e DISPLAY=$DISPLAY \
#		keepass2 "$@"
#
# ISSUES:
#	# 'Gtk: cannot open display: :0'
#	Try to set 'DISPLAY=your_host_ip:0' or run 'xhost +' on your host.
#	(see: https://stackoverflow.com/questions/28392949/running-chromium-inside-docker-gtk-cannot-open-display-0)
#

FROM debian:sid-slim
LABEL maintainer "Christian Koep <christiankoep@gmail.com>"

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && apt-get install -y \
	keepass2 \
	xdotool \
	mono-mcs \
	--no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

ENTRYPOINT ["/usr/bin/keepass2"]
