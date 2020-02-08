FROM debian:bullseye-slim
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

RUN apt-get update && apt-get install -y \
	rt-tests \
	--no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

CMD [ "hackbench" ]
