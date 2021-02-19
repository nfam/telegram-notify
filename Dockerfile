FROM golang:1.16 as builder
COPY . $GOPATH/src/telegram-notify
WORKDIR $GOPATH/src/telegram-notify
RUN go build -tags=netgo -ldflags '-s -w'

FROM alpine:3.11 as alpine
RUN apk add -U --no-cache ca-certificates

FROM scratch
COPY --from=builder /go/src/telegram-notify/telegram-notify /telegram-notify
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
EXPOSE 8000
ENTRYPOINT ["/telegram-notify"]
CMD []
