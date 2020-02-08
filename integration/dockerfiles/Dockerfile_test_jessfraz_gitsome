# Run gitsome command line tool:
# https://github.com/donnemartin/gitsome
#
# Usage:
# 	docker run --rm -it \
# 		-v ${HOME}/.gitsomeconfig:/home/anon/.gitsomeconfig \
# 		-v ${HOME}/.gitsomeconfigurl:/home/anon/.gitsomeconfigurl \
#		r.j3ss.co/gitsome
#
FROM python:3.5-alpine

RUN apk add --no-cache \
	bash

RUN pip3 install gitsome

ENV HOME /home/anon
RUN adduser -S anon \
	&& chown -R anon $HOME

WORKDIR $HOME
USER anon

ENTRYPOINT ["gitsome"]
