FROM golang:1.23.5-alpine3.21 as builder
WORKDIR /NewZGalleryBot
RUN apk update && apk upgrade --available && sync && apk add --no-cache --virtual .build-deps ca-certificates
COPY . .
RUN go build -ldflags="-w -s" .
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /NewZGalleryBot/NewZGalleryBot /NewZGalleryBot
ENTRYPOINT ["/NewZGalleryBot"]
