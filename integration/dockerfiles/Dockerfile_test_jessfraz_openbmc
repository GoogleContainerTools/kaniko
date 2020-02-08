FROM debian:buster-slim

RUN apt-get update && apt-get install -y \
	bash \
	build-essential \
	ca-certificates \
	chrpath \
	curl \
	diffstat \
	gawk \
	git \
	libpixman-1-0 \
	libsdl1.2-dev \
	texinfo \
	&& rm -rf /var/lib/apt/lists/*

# Download the latest qemu build from openBMC's fork
RUN curl -sSL -o /usr/bin/qemu-system-arm https://openpower.xyz/job/openbmc-qemu-build-merge-x86/lastSuccessfulBuild/artifact/qemu/arm-softmmu/qemu-system-arm \
	&& chmod +x /usr/bin/qemu-system-arm

# Download the latest romulus image
RUN mkdir -p /usr/src/openbmc
RUN curl -sSL -o /usr/src/openbmc/obmc-phosphor-image-romulus.static.mtd https://openpower.xyz/job/openbmc-build/distro=ubuntu,label=builder,target=romulus/lastSuccessfulBuild/artifact/deploy/images/romulus/obmc-phosphor-image-romulus.static.mtd

#ENV OPENBMC_VERSION 2.7.0

#RUN git clone --depth 1 --branch "${OPENBMC_VERSION}" https://github.com/openbmc/openbmc /usr/src/openbmc

#WORKDIR /usr/src/openbmc

#ENV TEMPLATECONF=meta-ibm/meta-palmetto/conf

#RUN bash ./openbmc-env \
#	&& bitbake obmc-phosphor-image

ENTRYPOINT ["qemu-system-arm",  "-m", "256", "-M", "romulus-bmc", "-nographic", "-drive", "file=/usr/src/openbmc/obmc-phosphor-image-romulus.static.mtd,format=raw,if=mtd", "-net", "nic", "-net", "user,hostfwd=:127.0.0.1:2222-:22,hostfwd=:127.0.0.1:2443-:443,hostname=qemu"]
