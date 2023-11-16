FROM golang:1.20-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /app/bin/bulk-loagen .

FROM alpine

COPY --from=builder /app/bin/bulk-loagen /

ENTRYPOINT ["/bulk-loagen"]
