FROM gcr.io/google-appengine/debian9:latest
RUN apt-get update && apt-get install -y curl
RUN echo "hey" > /etc/foo
RUN echo "baz" > /etc/foo
RUN echo "baz" > /etc/foo2

COPY ./pkg/image/ ./pkg/env/ foo/
COPY ./pkg/*/stor*.go ./pkg/env /storage/
COPY ./pkg/image/proxy*.go hello.go

ENV hello helloooo