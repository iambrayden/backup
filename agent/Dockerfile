FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.work go.work.sum* ./
COPY pkg/ pkg/
COPY agent/ agent/
RUN cd agent && go build -o /backup-agent ./cmd/agent

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /backup-agent /usr/local/bin/backup-agent
ENTRYPOINT ["backup-agent"]
