# docker build -t go-search-engine-v2 .
# docker run -dit --name go-search-engine-v2 -p 7004:7004 go-search-engine-v2

FROM golang:1.26 AS builder

RUN apt-get update && apt-get install bash && apt-get install -y --no-install-recommends ca-certificates

WORKDIR /app
COPY . .
RUN go mod tidy

WORKDIR /app
RUN go build -o go-search-engine-v2 -ldflags '-linkmode external -w -extldflags "-static"'

FROM alpine

WORKDIR /app
COPY --from=builder /app/go-search-engine-v2 .
COPY --from=builder /app/application/infrastructure/security/policy/policy.cedar /app/application/infrastructure/security/policy/policy.cedar
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

CMD ["/app/go-search-engine-v2"]