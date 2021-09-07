FROM golang:latest as build

LABEL maintainer "jxsl13@gmail.com"
WORKDIR /build
COPY . .
RUN go get -d && \
    CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-w -extldflags "-static"' -o econ_vpn_detection .


FROM alpine:3 as run

# enable OFFLINE in order to disable all of the 
# online VPN detection api usage below.
# this solely uses the redis database and 
# any provied blacklist/whitelist
ENV OFFLINE="false"
ENV IPHUB_TOKEN=""
ENV PROXYCHECK_TOKEN=""
ENV IPTEOH_ENABLED="false"
# 60 % of the above apis must declare a joining IP as VPN in order for it to be permanently added to the redis database
ENV PERMABAN_THRESHOLD="0.6"

ENV REDIS_ADDRESS="redis:6379"
ENV REDIS_PASSWORD=""
ENV REDIS_DB_VPN="0"
# n econ addresses delimited by a whitespace
ENV ECON_ADDRESSES=""
# a single password for all or n passwords for 
# each econ address above. Passwords are delimited 
# by a single whitespace.
ENV ECON_PASSWORDS=""
ENV VPN_BAN_REASON="VPN"
ENV VPN_BAN_DURATION="1h"
# the server that is monitored is a zCatch 0.7 server
ENV LOGGING_FORMAT="Vanilla"
# whether to ban players joining with a server's IP
ENV PROXY_DETECTION_ENABLED="false"
# how ften to fetch the IP list
ENV PROXY_UPDATE_INTERVAL="1m"

ENV PROXY_BAN_DURATION="24h"
ENV PROXY_BAN_REASON="proxy connection"

WORKDIR /app
COPY --from=build /build/econ_vpn_detection .
VOLUME ["/data/whitelists", "/data/blacklists"]
ENTRYPOINT ["/app/econ_vpn_detection"]
