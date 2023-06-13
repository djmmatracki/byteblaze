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

FROM ubuntu:20.04


COPY --from=builder /build/byteblaze /app/byteblaze

RUN mkdir -p /var/byteblaze

ENTRYPOINT ["/app/byteblaze"]
