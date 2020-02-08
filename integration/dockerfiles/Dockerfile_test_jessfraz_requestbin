FROM python:2-alpine

RUN apk add --no-cache --virtual .build-deps \
		build-base \
		git \
	&& git clone --depth 1 https://github.com/Runscope/requestbin /src \
    && pip install -r /src/requirements.txt \
	&& pip install --no-cache-dir  \
		gevent \
		gunicorn \
    && rm -rf ~/.pip/cache \
	&& apk del .build-deps

WORKDIR /src

CMD ["gunicorn", "-b", "0.0.0.0:8080", "requestbin:app", "-k", "gevent"]
