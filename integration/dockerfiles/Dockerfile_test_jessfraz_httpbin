# Run HTTPBin in a container
#
# USAGE
# 
# docker run -d \
#	-p 8080:8080 \
#	--name httpbin \
#	jess/httpbin
#

FROM python:3-alpine

RUN apk add --no-cache --virtual .build-deps \
		build-base \
		libffi-dev \
	&& pip3 install --no-cache-dir  \
		gevent \
		gunicorn \
		httpbin \
	&& apk del .build-deps

CMD ["gunicorn", "-b", "0.0.0.0:8080", "httpbin:app", "-k", "gevent"]
