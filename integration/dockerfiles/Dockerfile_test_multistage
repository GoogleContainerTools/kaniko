FROM gcr.io/google-appengine/debian9@sha256:f0159d14385afcb58a9b2fa8955c0cb64bd3abc365e8589f8c2dd38150fbfdbe as base
COPY . .

FROM scratch as second
ENV foopath context/foo
COPY --from=0 $foopath context/b* /foo/

FROM second as third
COPY --from=base /context/foo /new/foo

FROM base as fourth
# Make sure that we snapshot intermediate images correctly
RUN date > /date
ENV foo bar

# This base image contains symlinks with relative paths to whitelisted directories
# We need to test they're extracted correctly
FROM fedora@sha256:c4cc32b09c6ae3f1353e7e33a8dda93dc41676b923d6d89afa996b421cc5aa48

FROM fourth
ARG file
COPY --from=second /foo ${file}
COPY --from=gcr.io/google-appengine/debian9@sha256:00109fa40230a081f5ecffe0e814725042ff62a03e2d1eae0563f1f82eaeae9b /etc/os-release /new
