FROM alpine:3.11 as builder

RUN mkdir -p /myapp/somedir \
 && touch /myapp/somedir/somefile \
 && chown 123:123 /myapp/somedir \
 && chown 321:321 /myapp/somedir/somefile

FROM alpine:3.11
COPY --from=builder /myapp /myapp
RUN printf "%s\n" \
      "0 0 /myapp/" \
      "123 123 /myapp/somedir" \
      "321 321 /myapp/somedir/somefile" \
      > /tmp/expected \
 && stat -c "%u %g %n" \
      /myapp/ \
      /myapp/somedir \
      /myapp/somedir/somefile \
      > /tmp/got \
 && diff -u /tmp/got /tmp/expected
