FROM r.j3ss.co/bcc
MAINTAINER Jessica Frazelle <jess@linux.com>

ENV PATH /usr/share/bcc/tools:$PATH

# Add non-free apt sources
RUN sed -i "s#deb http://deb.debian.org/debian buster main#deb http://deb.debian.org/debian buster main contrib non-free#g" /etc/apt/sources.list

RUN apt-get update && apt-get install -y \
    ca-certificates \
	clang \
	curl \
	gcc \
	git \
	g++ \
    --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

# Build bpftrace
ENV BPFTRACE_VERSION v0.9.3
RUN git clone --depth 1 --branch "$BPFTRACE_VERSION" https://github.com/iovisor/bpftrace.git /usr/src/bpftrace \
	&& ( \
		cd /usr/src/bpftrace \
		&& mkdir build \
		&& cd build \
		&& cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/usr .. \
		&& make -j8 \
		&& make install \
	) \
	&& rm -rf /usr/src/bpftrace

ENTRYPOINT ["bpftrace"]
