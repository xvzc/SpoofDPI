FROM golang:alpine AS builder
WORKDIR /go
RUN go install -ldflags '-w -s -extldflags "-static"' -tags timetzdata github.com/xvzc/SpoofDPI/cmd/spoofdpi@latest

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/spoofdpi /
ENTRYPOINT ["/spoofdpi"]
