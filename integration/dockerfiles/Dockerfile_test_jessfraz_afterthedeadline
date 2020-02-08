FROM openjdk:8-alpine
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

RUN apk add --no-cache \
	ca-certificates \
	curl \
	tar

ENV LANG C.UTF-8
# https://open.afterthedeadline.com/download/download-source-code/
ENV ATD_VERSION 081310

RUN curl -sSL "http://www.polishmywriting.com/download/atd_distribution${ATD_VERSION}.tgz" -o /tmp/atd.tar.gz \
	&& mkdir -p /usr/src/atd \
	&& tar -xzf /tmp/atd.tar.gz -C /usr/src/atd --strip-components 1 \
	&& rm /tmp/atd.tar.gz*

WORKDIR /usr/src/atd
EXPOSE 1049

ENTRYPOINT [ "sh", "-c", "/usr/src/atd/run.sh" ]
