FROM golang:1.26.0-alpine3.23 AS builder
WORKDIR /NewZGalleryBot
RUN apk add --no-cache ca-certificates
COPY . .
RUN go build -trimpath -ldflags="-w -s" .
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /NewZGalleryBot/NewZGalleryBot /NewZGalleryBot
ENTRYPOINT ["/NewZGalleryBot"]
