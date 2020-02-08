# docker run -d --rm \
# 	-v /tmp/.X11-unix:/tmp/.X11-unix \
# 	-e DISPLAY=unix\$DISPLAY \
#	--device /dev/video0 \
#	--device /dev/snd:/dev/snd \
#	--device /dev/dri \
#	-v /dev/shm:/dev/shm \
#	jess/zoom-us

FROM debian:sid-slim

ENV DEBIAN_FRONTEND noninteractive

# Dependencies for the client .deb
RUN apt-get update && apt-get install -y \
	ca-certificates \
	curl \
	desktop-file-utils \
	ibus \
	ibus-gtk \
	lib32z1 \
	libx11-6 \
	libasound2-dev \
	libegl1-mesa \
	libxcb-shm0 \
	libglib2.0-0 \
	libgl1-mesa-glx \
	libxrender1 \
	libxcomposite1 \
	libxslt1.1 \
	libgstreamer1.0-dev \
	libgstreamer-plugins-base1.0-dev \
	libxi6 \
	libsm6 \
	libfontconfig1 \
	libpulse0 \
	libsqlite3-0 \
	libxcb-shape0 \
	libxcb-xfixes0 \
	libxcb-randr0 \
	libxcb-image0 \
	libxcb-keysyms1 \
	libxcb-xtest0 \
	libxtst6 \
	libnss3 \
	libxss1 \
	sudo \
	--no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

ARG ZOOM_URL=https://zoom.us/client/latest/zoom_amd64.deb

#install zoom
RUN curl -sSL $ZOOM_URL -o /tmp/zoom_setup.deb \
	&& dpkg -i /tmp/zoom_setup.deb \
	&& apt-get -f install \
	&& rm /tmp/zoom_setup.deb \
	&& rm -rf /var/lib/apt/lists/*

WORKDIR /usr/bin
CMD [ "./zoom" ]
