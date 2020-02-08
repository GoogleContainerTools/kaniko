# ncmpc is a fully featured MPD client
# which runs in a terminal (using ncurses)
#
# docker run --rm -it \
# 	-v /etc/localtime:/etc/localtime:ro \
#	--link mpd:mpd \
#	jess/ncmpc
#
FROM debian:sid-slim
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

RUN apt-get update && apt-get install -y \
	ncmpc \
	--no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

ENTRYPOINT [ "ncmpc" ]
