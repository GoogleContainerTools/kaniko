FROM busybox@sha256:1bd6df27274fef1dd36eb529d0f4c8033f61c675d6b04213dd913f902f7cafb5
ADD context/tars /tmp/tars
RUN stat /bin/sh
RUN mv /tmp/tars /foo
RUN echo "hi"
