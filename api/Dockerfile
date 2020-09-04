FROM golang:1.15-alpine

RUN apk add bash ca-certificates git gcc g++ libc-dev

RUN apk update \
    && apk upgrade \
    && apk add --no-cache \
    # Packages to install:
    bash \
    git \
    gcc \
    g++ \
    libc-dev \
    ca-certificates \
    # Update certificates
    && update-ca-certificates 2>/dev/null || true
WORKDIR /opt
ENV GO111MODULE=on

# Download all dependencies.
# (Dependencies will be cached if the go.mod and go.sum files are not changed)
COPY go.mod go.sum ./
RUN go mod download

# Build in separate step
COPY . ./
RUN go build -o main .

EXPOSE 8080
CMD ["./main"]
