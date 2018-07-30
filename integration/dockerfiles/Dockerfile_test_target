FROM gcr.io/distroless/base:latest as base
COPY . .

FROM scratch as second
ENV foopath context/foo
COPY --from=0 $foopath context/b* /foo/

FROM base
ARG file
COPY --from=second /foo $file
