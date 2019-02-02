ARG REGISTRY=gcr.io
ARG REPO=google-appengine
ARG WORD=hello
ARG W0RD2=hey

FROM ${REGISTRY}/${REPO}/debian9 as stage1

# Should evaluate WORD and create /tmp/hello
ARG WORD
RUN touch /${WORD}

FROM ${REGISTRY}/${REPO}/debian9

COPY --from=stage1 /hello /tmp

# /tmp/hey should not get created without the ARG statement
# Use -d 0 to force a time change because of stat resolution
RUN touch -d 0 /tmp/${WORD2}
