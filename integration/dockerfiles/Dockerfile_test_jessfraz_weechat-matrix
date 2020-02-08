# Usage:
# 	Building
# 		docker build -t weechat-matrix .
#	Running (no saved state)
# 		docker run -it \
#			-v /etc/localtime:/etc/localtime:ro \ # for your time
# 			weechat-matrix
# 	Running (saved state)
# 		docker run -it \
#			-v /etc/localtime:/etc/localtime:ro \ # for your time
# 			-v "${HOME}/.weechat:/home/user/.weechat" \
# 			weechat-matrix
#
FROM alpine:latest

RUN apk add --no-cache \
	build-base \
	ca-certificates \
	git \
	libffi-dev \
	libressl-dev \
	olm-dev \
	python \
	python-dev \
	py2-pip \
	weechat \
	weechat-perl \
	weechat-python \
	--repository https://dl-4.alpinelinux.org/alpine/edge/testing

ENV HOME /home/user

RUN adduser -S user -h $HOME \
	&& chown -R user $HOME \
	&& cd $HOME \
	&& git clone https://github.com/poljar/weechat-matrix.git \
	&& cd weechat-matrix \
	&& pip install -r requirements.txt \
	&& pip install websocket-client \
	&& make install \
	&& chown -R user $HOME

WORKDIR $HOME
USER user

CMD [ "weechat" ]
