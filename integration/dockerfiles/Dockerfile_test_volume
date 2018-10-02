FROM gcr.io/google-appengine/debian9@sha256:1d6a9a6d106bd795098f60f4abb7083626354fa6735e81743c7f8cfca11259f0
RUN mkdir /foo
RUN echo "hello" > /foo/hey
VOLUME /foo/bar /tmp /qux/quux
ENV VOL /baz/bat
VOLUME ["${VOL}"]
RUN echo "hello again" > /tmp/hey
