FROM docker.io/debian:bullseye-slim as base
FROM base as build
COPY ["top1", "/tmp/top1"]
RUN \
  set -eu; \
  cp /tmp/top1 /usr/local/bin/top1; \
  chown root:root /usr/local/bin/top1; \
  chmod u=rxs,go=rx /usr/local/bin/top1; \
  ls -lh /usr/local/bin/top1
FROM base as final
COPY --from=build ["/usr/local/bin/top1", "/usr/local/bin/"]
LABEL \
  description="Testing setuid behavior in Kaniko"
