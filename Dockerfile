FROM golang:1.22-alpine AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w -X main.Version=docker" \
    -o prismbox .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates iptables ip6tables

COPY --from=builder /build/prismbox /usr/local/bin/prismbox

EXPOSE 9090 12345

VOLUME /root/.prismbox

ENTRYPOINT ["prismbox"]
CMD ["--listen", ":9090"]
