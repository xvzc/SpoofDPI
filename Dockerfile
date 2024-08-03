FROM golang:alpine

WORKDIR /go

RUN go install github.com/xvzc/SpoofDPI/cmd/spoof-dpi@latest

ENV ADDRESS=0.0.0.0

ENV DNS=1.1.1.1

ENV PORT=8080

ENV DEBUG=false

ENV NO_BANNER=true

ENV TIMEOUT=500

ENV URLS=

ENV PATTERN=

ENV WINDOW_SIZE=0

EXPOSE 8080

CMD ["/bin/sh", "-c", "/go/bin/spoof-dpi -addr=${ADDRESS} -debug=${DEBUG} -dns-addr=${DNS} -port=${PORT} -no-banner=${NO_BANNER} -timeout=${TIMEOUT} -window-size=${WINDOW_SIZE} $(echo \"${URLS}\" | tr -d ' ' | tr ',' '\n' | sed -e 's/^/-url=/') -pattern ${PATTERN}"]
