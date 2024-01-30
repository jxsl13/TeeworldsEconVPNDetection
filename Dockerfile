FROM golang:latest as build

LABEL maintainer "jxsl13@gmail.com"
WORKDIR /build
COPY . .
RUN go get -d && \
    CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-w -extldflags "-static"' -o econ_vpn_detection .


FROM alpine:3 as run


WORKDIR /app
COPY --from=build /build/econ_vpn_detection .
VOLUME ["/data/whitelists", "/data/blacklists"]
ENTRYPOINT ["/app/econ_vpn_detection"]
