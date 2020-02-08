FROM python:3-alpine

RUN apk add --no-cache \
	ca-certificates \
	bash \
	gfortran \
	lapack \
	openjdk8-jre-base \
	py3-numpy \
	py3-scipy

# Install the requirements
RUN set -x \
	&& apk add --no-cache --virtual .build-deps \
		build-base \
		git \
		lapack-dev \
		libffi-dev \
		openssl-dev \
	&& ln -s /usr/include/locale.h /usr/include/xlocale.h \
	&& git clone --depth 1 https://github.com/sarahsharp/foss-heartbeat.git /usr/src/foss-heartbeat \
	&& ( \
		cd /usr/src/foss-heartbeat \
		&& cat requirements.txt | grep -v numpy | grep -v scipy | tee  requirements.txt \
		&& pip install -r requirements.txt \
		&& pip install statistics \
	) \
	&& apk del .build-deps

WORKDIR /usr/src/foss-heartbeat
