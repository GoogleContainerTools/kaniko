FROM gcr.io/google-appengine/debian9@sha256:1d6a9a6d106bd795098f60f4abb7083626354fa6735e81743c7f8cfca11259f0
VOLUME /foo1
RUN echo "hello" > /foo1/hello
WORKDIR /foo1/bar
ADD context/foo /foo1/foo
COPY context/foo /foo1/foo2
RUN mkdir /bar1
VOLUME /foo2
VOLUME /foo3
RUN echo "bar2"
VOLUME /foo4
RUN mkdir /bar3
VOLUME /foo5
