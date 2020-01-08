FROM golang:1.10-alpine AS builder
ARG BUILDTAGS=""
COPY . /go/src/github.com/moby/buildkit
RUN CGO_ENABLED=0 go build -o /dockerfile-frontend -tags "$BUILDTAGS" --ldflags '-extldflags "-static"' github.com/moby/buildkit/frontend/dockerfile/cmd/dockerfile-frontend

FROM scratch
COPY --from=builder /dockerfile-frontend /bin/dockerfile-frontend
ENTRYPOINT ["/bin/dockerfile-frontend"]