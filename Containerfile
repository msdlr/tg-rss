FROM golang:1.26-alpine AS builder
WORKDIR /app
RUN apk add --no-cache gcc musl-dev sqlite-dev
COPY . .
RUN CGO_ENABLED=1 go build

# Final stage
FROM alpine:3.19

RUN apk add --no-cache sqlite-libs

RUN addgroup -g 1001 -S tgrss && \
    adduser -u 1001 -S tgrss -G tgrss
WORKDIR /app

COPY --from=builder /app/tg-rss .
RUN chown -R tgrss:tgrss /app

USER tgrss

CMD ["./tg-rss"]