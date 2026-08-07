FROM golang:1.26-alpine AS builder
WORKDIR /app
RUN apk add --no-cache gcc musl-dev sqlite-dev git
COPY . .
RUN HEAD="$(git describe --tags)"; sed -i "s/HEAD/$HEAD/" info/info.go
RUN BUILD_DATE="$(git log -1 --date=format:'%Y/%m/%d %T' --format='%ad')"; sed -i "s|BUILD_DATE|$BUILD_DATE|" info/info.go
RUN CGO_ENABLED=1 go build

# Final stage
FROM alpine:3.24

RUN apk add --no-cache sqlite-libs

WORKDIR /app

COPY --from=builder /app/tg-rss .

CMD ["./tg-rss"]