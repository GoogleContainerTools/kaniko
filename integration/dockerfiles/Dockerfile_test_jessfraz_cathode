FROM debian:bullseye-slim
LABEL maintainer "Jessie Frazelle <jess@linux.com>"

RUN apt-get update && apt-get install -y \
	build-essential \
	ca-certificates \
	git \
	qmlscene \
	qt5-qmake \
	qt5-default \
	qtdeclarative5-dev \
	qml-module-qtquick-controls \
	qml-module-qtgraphicaleffects \
	qml-module-qtquick-dialogs \
	qml-module-qtquick-localstorage \
	qml-module-qtquick-window2 \
	--no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

RUN git clone --depth 1 --recursive https://github.com/Swordfish90/cool-retro-term.git /src
WORKDIR /src
RUN qmake \
	&& make

ENTRYPOINT [ "/src/cool-retro-term" ]
