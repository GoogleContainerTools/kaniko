FROM marketplace.gcr.io/google/ubuntu1804@sha256:4649ae6b381090fba6db38137eb05e03f44bf43c40149f734241c9f96aa0e001
ENV dir /tmp/dir/
ONBUILD RUN echo "onbuild" > /tmp/onbuild
ONBUILD RUN  mkdir $dir
ONBUILD RUN echo "onbuild 2" > ${dir}/onbuild2
ONBUILD WORKDIR /new/workdir
