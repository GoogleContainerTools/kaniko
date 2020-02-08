# Couchpotato in a container
#
# docker run -d \
# 	--restart always \
#	-p 5050:5050 \
# 	-v /etc/localtime:/etc/localtime:ro \
# 	-v /volumes/couchpotato:/data \
#	--link transmission:transmission \
# 	--name couchpotato \
# 	jess/couchpotato
#
FROM python:2-alpine
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

# machine parsable metadata, for https://github.com/pycampers/dockapt
LABEL "registry_image"="r.j3ss.co/couchpotato"
LABEL "docker_run_flags"="-d \
 	--restart always \
	-p 5050:5050 \
 	-v /etc/localtime:/etc/localtime:ro \
 	-v /volumes/couchpotato:/data \
	--link transmission:transmission \
 	--name couchpotato"

RUN apk add --no-cache \
	ca-certificates \
	gcc \
	git \
	libffi-dev \
	libxml2-dev \
	libxslt-dev \
	musl-dev \
	openssl-dev \
	&& rm -rf /var/lib/apt/lists/*

RUN pip install \
	lxml \
	pyopenssl

EXPOSE 5050

ENV COUCHPOTATO_VERSION master

RUN git clone https://github.com/RuudBurger/CouchPotatoServer.git /usr/src/couchpotato \
	&& ( \
		cd /usr/src/couchpotato \
		&& git checkout "${COUCHPOTATO_VERSION}" \
	)

WORKDIR /usr/src/couchpotato

ENTRYPOINT [ "python", "CouchPotato.py", "--debug" ]
CMD [ "--data_dir", "/data" ]
