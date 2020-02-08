FROM debian:bullseye-slim
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

# Install bash and other deps so we have them.
RUN apt-get update && apt-get install -y \
		bash \
		bzip2 \
		ca-certificates \
		libdbus-glib-1-2 \
		libgtk-3-0 \
		libx11-xcb1 \
		libxt6 \
		tar \
		wget \
		--no-install-recommends \
		&& rm -rf /var/lib/apt/lists/*

RUN wget "https://raw.githubusercontent.com/StevenBlack/hosts/master/alternates/fakenews-gambling-porn-social/hosts" -O /etc/hosts

# Create user and change ownership
RUN addgroup --gid 666 browsh \
	&& adduser --uid 666 --home /home/browsh --ingroup browsh browsh

WORKDIR /home/browsh
USER browsh

RUN mkdir -p /home/browsh/bin

ENV PATH="/bin:/usr/bin:/usr/local/bin:/home/browsh/bin:${PATH}"

# Install firefox.
ENV FIREFOX_VERSION 60.0
RUN set -x \
	&& wget "https://ftp.mozilla.org/pub/firefox/releases/${FIREFOX_VERSION}/linux-x86_64/en-US/firefox-${FIREFOX_VERSION}.tar.bz2" -O /tmp/firefox.tar.bz2 \
	&& ( \
		cd /tmp \
		&& bzip2 -d /tmp/firefox.tar.bz2 \
		&& tar -xf /tmp/firefox.tar -C /home/browsh/bin/ --strip-components 1 \
	) \
	&& rm -rf /tmp/firefox* \
	&& firefox --version

# Install browsh.
ENV BROWSH_VERSION 1.6.4
RUN wget "https://github.com/browsh-org/browsh/releases/download/v${BROWSH_VERSION}/browsh_${BROWSH_VERSION}_linux_amd64" -O /home/browsh/bin/browsh \
	&& chmod a+x /home/browsh/bin/browsh

# Firefox behaves quite differently to normal on its first run, so by getting
# that over and done with here when there's no user to be dissapointed means
# that all future runs will be consistent.
RUN TERM=xterm browsh & \
		pidsave=$!; \
		sleep 10; kill $pidsave || true;

ENTRYPOINT [ "browsh" ]
