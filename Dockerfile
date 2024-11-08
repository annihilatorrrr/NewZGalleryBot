FROM golang:1.23.3-alpine3.20 as builder
WORKDIR /NewZGalleryBot
RUN apk update && apk upgrade --available && sync && apk add --no-cache --virtual .build-deps
COPY . .
RUN go build -ldflags="-w -s" .
FROM alpine:3.20.3
RUN apk update && apk upgrade --available && sync
COPY --from=builder /NewZGalleryBot/NewZGalleryBot /NewZGalleryBot
ENTRYPOINT ["/NewZGalleryBot"]
