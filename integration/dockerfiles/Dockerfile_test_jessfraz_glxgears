# To use:
#	docker run -v /tmp/.X11-unix:/tmp/.X11-unix \
#		-e DISPLAY=unix$DISPLAY \
#		--device /dev/dri \
#		jess/glxgears
#

# Base docker image
FROM debian:bullseye-slim
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

# Install all the things
RUN apt-get update && apt-get install -y \
	mesa-utils \
	--no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

ENV LIBGL_DEBUG verbose

ENTRYPOINT [ "glxgears" ]
