#FROM scratch
FROM alpine:latest

ADD simple-canary /app/
CMD ["/app/simple-canary", "--config", "/config/config.yaml"]
