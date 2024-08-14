FROM golang:alpine as builder

WORKDIR /go

RUN go install github.com/xvzc/SpoofDPI/cmd/spoof-dpi@latest

FROM alpine:latest

WORKDIR /

COPY --from=builder /go/bin/spoof-dpi .

EXPOSE 8080

ENTRYPOINT ["./spoof-dpi"]
