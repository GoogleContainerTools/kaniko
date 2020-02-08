# Run Rainbowstream in a container
#
# docker run -it --rm \
#	-v /etc/localtime:/etc/localtime:ro \
#	-v $HOME/.rainbow_oauth:/root/.rainbow_oauth \ # mount config files
#	-v $HOME/.rainbow_config.json:/root/.rainbow_config.json \
#	--name rainbowstream \
#	jess/rainbowstream
#
FROM python:2-alpine
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

RUN apk --no-cache add \
	build-base \
	ca-certificates \
	freetype \
	freetype-dev \
	openjpeg-dev \
	zlib-dev

RUN USER=root pip install \
	pillow==2.8.0 \
	rainbowstream

ENTRYPOINT [ "rainbowstream" ]
