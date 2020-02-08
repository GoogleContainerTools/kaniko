# Run Fontforge in a container
#
# state=$HOME
# mkdir -p $state/fontforge
# docker run --rm \
#     -v /tmp/.X11-unix:/tmp/.X11-unix \
#     -e DISPLAY=unix$DISPLAY \
#     -v $state/fontforge:/home/fontforge \
#     --name fontforge \
#     fghj/fontforge

# Base docker image
FROM ubuntu:16.04
LABEL maintainer "Axel Svensson <foss@axelsvensson.com>"

RUN  apt-get update \
  && apt-get install -y \
     software-properties-common \
     --no-install-recommends \
  && add-apt-repository ppa:fontforge/fontforge \
  && apt-get update \
  && apt-get install -y \
     fontforge \
     --no-install-recommends \
  && rm -rf /var/lib/apt/lists/*

ENV HOME /home/fontforge
RUN useradd --create-home --home-dir $HOME fontforge
WORKDIR $HOME
USER fontforge
CMD [ "fontforge" ]

