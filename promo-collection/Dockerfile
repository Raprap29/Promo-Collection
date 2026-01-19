FROM golang:1.24-alpine3.20 AS builder

RUN apk add --no-cache gcc musl-dev libc-dev librdkafka-dev

WORKDIR /app

ENV CGO_ENABLED=1
ENV GOFLAGS="-tags=dynamic"
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /app/promo-collection ./cmd/promo-collection
RUN ls

FROM alpine:3.21

RUN addgroup -g 1000 appuser && adduser -u 1000 -G appuser -D appuser && apk add --no-cache librdkafka && apk upgrade --no-cache

WORKDIR /app

COPY --from=builder --chown=appuser:appuser /app/promo-collection /app/promo-collection
COPY --from=builder --chown=appuser:appuser /app/configs/config.yaml /app/configs/config.yaml

USER appuser

ENTRYPOINT ["/app/promo-collection"]
