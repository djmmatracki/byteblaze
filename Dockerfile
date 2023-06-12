FROM golang:1.19-alpine3.17 AS builder

RUN apk add --virtual build-dependencies build-base gcc wget git

# Move to working directory (/build).
WORKDIR /build

COPY go.mod ./
RUN go mod download

# Copy the code into the container.
COPY . .

# Set necessary environment variables needed for our image and build the API server.
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -o byteblaze cmd/main.go

# DEBUG: install dlv debugger
RUN go install github.com/go-delve/delve/cmd/dlv@latest

FROM ubuntu:20.04

# DEBUG: we have to copy the source code to debug anyway ...
COPY . .
COPY --from=builder /build/byteblaze /app/byteblaze
# DEBUG: copy the debugger
COPY --from=builder /go/bin/dlv dlv
COPY .torrent /app/.torrent

RUN mkdir -p /var/byteblaze

# ENTRYPOINT ["/app/byteblaze"]
# DEBUG: run the entry point through a debugger
ENTRYPOINT [ "/dlv" , "--listen=:40000", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "/app/byteblaze"]]