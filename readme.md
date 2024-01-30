
# Teeworlds VPN Detection & Distributed Banserver (written in Go)

This application connects to teeworlds servers via its configured external console (econ).
It reads every logged line and checks for joining players.
The joining player's IP is then compared to the redis cache.
Does the cache not contain the IP, currently three VPN detection APIs are used to determine whether the player's IP is a VPN or not.
60% of these three APIs need to detect the IP as VPN in order for the application to actually ban the player and cache his VPN IP in the redis cache as such.

## Requirements

### Docker

```
make start

makes stop
```

### Redis server for caching of IPs

This application requires a running redis database that can be used as cache for IPs.
The application caches non-VPN IPs in the redis database for one week.
VPN IPs are saved forever in order not to hit the free rate limit of the used APIs too fast.

#### Debian & Ubuntu

On Linux it is usually started automatically after its installation.

```shell

sudo apt install redis-server
```

#### macOS

On macOS you need to manually start it with `redis-server`.

```shell
brew install redis
```

## Building

```shell
go build .
```

## Running

You may use a `.env` config file for configuring the application.
Use `sample.env` as reference or check out the help of the application.

```shell
You can automatically add all of those IPs and the IPs from the subnets to your redis cache.
In order for such a file to be parsed, you can pass it on startup to the application like this:

Run the econ log parser with the VPN detection.
```text
Environment variables:
  TWVPN_IPHUB_TOKEN
  TWVPN_PROXYCHECK_TOKEN
  TWVPN_IPTEOH_ENABLED          (default: "false")
  TWVPN_REDIS_ADDRESS           (default: "localhost:6379")
  TWVPN_REDIS_PASSWORD
  TWVPN_REDIS_DB_VPN            (default: "0")
  TWVPN_ECON_ADDRESSES
  TWVPN_ECON_PASSWORDS
  TWVPN_RECONNECT_DELAY         (default: "10s")
  TWVPN_RECONNECT_TIMEOUT       (default: "24h0m0s")
  TWVPN_VPN_BAN_DURATION        (default: "5m0s")
  TWVPN_VPN_BAN_REASON          (default: "VPN")
  TWVPN_OFFLINE                 (default: "false")
  TWVPN_PERMA_BAN_THRESHOLD     (default: "0.6")
  TWVPN_IP_WHITELIST
  TWVPN_IP_BLACKLIST

Usage:
  TeeworldsEconVPNDetectionGo [flags]
  TeeworldsEconVPNDetectionGo [command]

Available Commands:
  add         add ips to the database (blacklist)
  completion  Generate completion script
  help        Help about any command
  remove      remove ips from the database (whitelist)

Flags:
  -c, --config string                .env config file path (or via env variable TWVPN_CONFIG)
      --econ-addresses string
      --econ-passwords string
  -h, --help                         help for TeeworldsEconVPNDetectionGo
      --ip-blacklist string
      --ip-whitelist string
      --iphub-token string
      --ipteoh-enabled
      --offline
      --perma-ban-threshold float     (default 0.6)
      --proxycheck-token string
      --reconnect-delay duration      (default 10s)
      --reconnect-timeout duration    (default 24h0m0s)
      --redis-address string          (default "localhost:6379")
      --redis-db-vpn int
      --redis-password string
      --vpn-ban-duration duration     (default 5m0s)
      --vpn-ban-reason string         (default "VPN")

Use "TeeworldsEconVPNDetectionGo [command] --help" for more information about a command.
```

Add ips to the database (blacklist)
```shell
Environment variables:
  TWVPN_REDIS_ADDRESS      (default: "localhost:6379")
  TWVPN_REDIS_PASSWORD
  TWVPN_REDIS_DB_VPN       (default: "0")

Usage:
  TeeworldsEconVPNDetectionGo add [flags]

Flags:
  -c, --config string           .env config file path (or via env variable TWVPN_CONFIG)
  -h, --help                    help for add
      --redis-address string     (default "localhost:6379")
      --redis-db-vpn int
      --redis-password string
```

Remove ips from the database (whitelist)
```shell
Environment variables:
  TWVPN_REDIS_ADDRESS      (default: "localhost:6379")
  TWVPN_REDIS_PASSWORD
  TWVPN_REDIS_DB_VPN       (default: "0")

Usage:
  TeeworldsEconVPNDetectionGo remove [flags]

Flags:
  -c, --config string           .env config file path (or via env variable TWVPN_CONFIG)
  -h, --help                    help for remove
      --redis-address string     (default "localhost:6379")
      --redis-db-vpn int
      --redis-password string
```


## Add/Remove IPs from IPv4 text file to/from the Redis database

In order for this to work, you need to have a properly configured setup with a `.env` file.
Given a file with conents like:

```text
1.236.132.203

# this adds/removes 2.56.0.1 through 2.56.255.254 to the database
2.56.92.0/16

# this adds/removes the IPs 2.56.140.1 through 2.56.140.254 from the database
2.56.140.0/24

123.0.0.1 # add any custom ban reason
123.0.0.1/24 # also add any custom ban reason, # followed by text

213.182.158.200 - 213.182.158.203 # reason (excluding the upper boundary IP, IPs ending with 0 or 255)

```

Due to the underlying *goripr* library the insertion of those IP ranges is pretty fast and storage efficient.

After all of the IPs have been parsed and added to the cache, the application shuts down.
You need to restart it without the flag in order to have the econ VPN detection behavior.

## Note

Currently no IPv6 support.
Add a `# ban reason` behind the IP or behind the IP range to add a custom ban reason.

## Troubleshooting

### EOF error

The Teeworlds server banned you for attempting to login too any times or for connecting too often.
Possible solution is to restart the game server.
This should not be an issue, as the detector attempts to reconnect to the server.
