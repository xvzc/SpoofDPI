FROM golang:alpine as builder

WORKDIR /go

RUN go install github.com/xvzc/SpoofDPI/cmd/spoofdpi@latest

FROM alpine:latest

WORKDIR /

COPY --from=builder /go/bin/spoofdpi .

ENTRYPOINT ["./spoofdpi"]
