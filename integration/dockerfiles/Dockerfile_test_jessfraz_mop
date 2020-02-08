# Run mop-tracker in a container
#
# docker run -it --rm \
# 	-v ~/.moprc:/root/.moprc \
# 	--name mop \
# 	r.j3ss.co/mop
#
FROM golang:alpine as builder

RUN apk --no-cache add \
	ca-certificates \
	git

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

RUN go get github.com/mop-tracker/mop/cmd/mop

FROM alpine:latest
COPY --from=builder /go/bin/mop /usr/bin/mop
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs
ENTRYPOINT [ "mop" ]
