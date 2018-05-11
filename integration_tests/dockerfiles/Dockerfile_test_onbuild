FROM gcr.io/kaniko-test/onbuild-base:latest
COPY context/foo foo
ENV dir /new/workdir/
ARG file
ONBUILD RUN echo "onbuild" > $file
ONBUILD RUN echo "onbuild 2" > ${dir}
ONBUILD WORKDIR /new/workdir
