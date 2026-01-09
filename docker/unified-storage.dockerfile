FROM nats:2.12.3-alpine3.22

RUN apk add --no-cache ca-certificates wget curl jq go git

RUN go install github.com/nats-io/natscli/nats@latest && \
  mv /root/go/bin/nats /usr/local/bin/nats && \
  apk del go git
