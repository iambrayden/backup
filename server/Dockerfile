FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.work go.work.sum* ./
COPY pkg/ pkg/
COPY server/ server/
RUN cd server && go build -o /server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
