# Run spotify windows app in a container with wine
#
# docker run --rm -it \
#	-v /etc/localtime:/etc/localtime:ro \
#	--cpuset-cpus 0 \
#	-v /tmp/.X11-unix:/tmp/.X11-unix  \
#	-e DISPLAY=unix$DISPLAY \
#	--device /dev/snd:/dev/snd \
#	--name spotify-wine \
#	jess/spotify-wine bash
#
FROM r.j3ss.co/wine
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

ADD https://download.scdn.co/SpotifySetup.exe /usr/src/SpotifySetup.exe

RUN echo "wine /usr/src/SpotifySetup.exe" > /root/.bash_history

CMD [ "bash" ]
