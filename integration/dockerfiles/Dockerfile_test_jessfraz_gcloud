FROM python:2-alpine

ENV PATH $PATH:/usr/src/google-cloud-sdk/bin
ENV CLOUD_SDK_DOCKER_VERSION 272.0.0

RUN set -x \
	&& apk add --no-cache --virtual .build-deps \
		curl \
		ca-certificates \
		tar \
	&& curl -sSL -o /tmp/google-cloud-sdk.tar.gz "https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-${CLOUD_SDK_DOCKER_VERSION}-linux-x86_64.tar.gz" \
	&& mkdir -p /usr/src/google-cloud-sdk \
	&& tar -xzf /tmp/google-cloud-sdk.tar.gz -C /usr/src/google-cloud-sdk --strip-components 1 \
	&& rm /tmp/google-cloud-sdk.tar.gz \
	&& ( \
		cd /usr/src/google-cloud-sdk \
		&& ./install.sh \
        	--usage-reporting false \
        	--path-update false \
	) \
	&& apk del .build-deps \
	&& gcloud components update \
	&& gcloud version

WORKDIR /root

ENTRYPOINT ["gcloud"]
