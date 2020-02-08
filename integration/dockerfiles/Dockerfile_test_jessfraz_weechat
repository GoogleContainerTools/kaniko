# Run weechat in a container
#
# docker run -it \
#	-v $HOME/.weechat/home/user/.weechat \
#	--name weechat \
#	jess/weechat
#

FROM alpine:latest

RUN apk add --no-cache \
	weechat \
	weechat-perl \
	weechat-python \
	python

ARG RUNTIME_UID
ENV RUNTIME_UID ${UID:-1000}
ENV HOME /home/user

RUN adduser -D user -u ${RUNTIME_UID} \
	&& chown -R user $HOME

WORKDIR $HOME
USER user

ENTRYPOINT [ "weechat" ]
