FROM golang:alpine as builder

WORKDIR /go
COPY . .
RUN go build -o spoof-dpi ./cmd/spoof-dpi/main.go

FROM alpine:latest

WORKDIR /

COPY --from=builder /go/bin/spoof-dpi .

ARG RUN_ARGS

ENTRYPOINT ["./spoof-dpi", "${RUN_ARGS}"]
