FROM debian:sid-slim
LABEL maintainer "Christian Koep <christiankoep@gmail.com>"

ENV ROUTERSPLOIT_VERSION v3.4.0

RUN apt-get update && apt-get install -y \
    git \
    python-requests \
    python-paramiko \
    python-pysnmp-common \
    python-bs4 \
    --no-install-recommends \
    && rm -rf /var/lib/apt/lists/* \
    && git clone --depth 1 --branch "${ROUTERSPLOIT_VERSION}" https://github.com/reverse-shell/routersploit /usr/bin/routersploit \
    && apt-get purge -y --auto-remove \
    git

WORKDIR "/usr/bin/routersploit/"
ENTRYPOINT [ "./rsf.py" ]
