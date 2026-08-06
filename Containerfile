FROM golang:1.26-alpine AS builder
WORKDIR /app
RUN apk add --no-cache gcc musl-dev sqlite-dev git
COPY . .
RUN HEAD="$(git log -1 --pretty=format:%h)"; sed -i "s/HEAD/$HEAD/" info/info.go
RUN BUILD_DATE="$(git log -1 --date=format:'%Y/%m/%d %T' --format='%ad')"; sed -i "s|BUILD_DATE|$BUILD_DATE|" info/info.go
RUN LATEST_TAG="$(git describe --tags --abbrev=0)"; sed -i "s/LATEST_TAG/$LATEST_TAG/" info/info.go
RUN SUBVERSION="$(git rev-list "$(git describe --tags --abbrev=0)"..HEAD --count)"; sed -i "s/SUBVERSION/$SUBVERSION/" info/info.go
RUN CGO_ENABLED=1 go build

# Final stage
FROM alpine:3.24

RUN apk add --no-cache sqlite-libs

WORKDIR /app

COPY --from=builder /app/tg-rss .

CMD ["./tg-rss"]