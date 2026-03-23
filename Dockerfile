FROM golang:1.23-bookworm AS builder

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /blackcat ./cmd/blackcat/

# Runtime image
FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /blackcat /usr/local/bin/blackcat

# Create non-root user
RUN useradd -m -s /bin/bash blackcat
USER blackcat
WORKDIR /home/blackcat

ENTRYPOINT ["blackcat"]
CMD ["serve"]
