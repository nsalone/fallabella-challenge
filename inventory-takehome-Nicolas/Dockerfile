FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o ingest ./cmd/ingest

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/ingest .
COPY data ./data

CMD ["./ingest"]
