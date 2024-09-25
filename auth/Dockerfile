FROM golang:1.22 AS local

WORKDIR /usr/src/app

RUN go install github.com/cosmtrek/air@v1.51.0

COPY auth .

RUN go mod download

CMD ["air"]

FROM golang:1.22 AS builder

WORKDIR /usr/src/app

COPY auth .

RUN go mod download

RUN mkdir -p /usr/src/app/tmp

# Build server executable
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /usr/src/app/tmp/server_process ./auth/main.go

FROM alpine:3.14 AS production

WORKDIR /usr/bin/app

# Install bash
RUN apk add --no-cache bash

# Create directories
RUN mkdir -p /usr/bin/app

COPY auth/.env /usr/bin/app/.env

# Copy server executable
COPY --from=builder /usr/src/app/tmp/server_process /usr/bin/app/server_process

# Set permissions
RUN chmod +x /usr/bin/app/server_process

CMD ["/usr/bin/app/server_process"]
#CMD ["tail", "-f", "/dev/null"]
