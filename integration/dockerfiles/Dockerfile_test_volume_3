FROM scratch as one
VOLUME /vol

FROM alpine@sha256:5ce5f501c457015c4b91f91a15ac69157d9b06f1a75cf9107bf2b62e0843983a as two
RUN mkdir /vol && echo hey > /vol/foo