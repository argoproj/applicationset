# Build the binary
FROM golang:1.14.1 as builder

WORKDIR /workspace
# Copy the go source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -a -o applicationset-controller main.go

# Use distroless as minimal base image to package the manager binary
FROM debian:10-slim
RUN apt-get update && apt-get upgrade -y && \
    apt-get install -y git-all && \
    rm -r /var/lib/apt/lists /var/cache/apt/archives
WORKDIR /
COPY --from=builder /workspace/applicationset-controller /usr/local/bin/