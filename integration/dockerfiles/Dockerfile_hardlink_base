FROM alpine@sha256:5ce5f501c457015c4b91f91a15ac69157d9b06f1a75cf9107bf2b62e0843983a AS stage1
RUN apk --no-cache add git
RUN rm /usr/bin/git && ln -s /usr/libexec/git-core/git /usr/bin/git
