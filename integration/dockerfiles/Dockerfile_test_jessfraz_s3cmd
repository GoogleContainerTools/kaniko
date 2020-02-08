# s3cmd in a container
#
# docker run --rm -it \
#	-e AWS_ACCESS_KEY \
#	-e AWS_SECRET_KEY \
#	-v $(pwd):/root/s3cmd-workspace
#	--name s3cmd \
#	jess/s3cmd
#
FROM debian:sid-slim
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

RUN apt-get update && apt-get install -y \
	ca-certificates \
	s3cmd \
	--no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

# Setup s3cmd config
RUN { \
		echo '[default]'; \
		echo 'access_key=$AWS_ACCESS_KEY'; \
		echo 'secret_key=$AWS_SECRET_KEY'; \
	} > ~/.s3cfg

ENV HOME /root
WORKDIR $HOME/s3cmd-workspace

ENTRYPOINT [ "s3cmd" ]
