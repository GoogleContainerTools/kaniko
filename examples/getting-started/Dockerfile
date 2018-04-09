FROM golang:1.10

WORKDIR /go/src/github.com/GoogleCloudPlatform/skaffold/examples/getting-started
CMD ["./app"]
COPY main.go .
RUN go build -o app main.go
