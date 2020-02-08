FROM alpine:latest
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

ENV HOME /home/gcalcli

RUN apk --no-cache add \
	python \
	python-dev \
	py2-pip \
	build-base \
	&& adduser -S gcalcli \
	&& chown -R gcalcli $HOME \
	&& pip install vobject parsedatetime gcalcli

WORKDIR $HOME
USER gcalcli

ENTRYPOINT [ "gcalcli" ]
