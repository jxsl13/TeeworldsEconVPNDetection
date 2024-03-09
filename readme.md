
# Teeworlds VPN Detection & Distributed Banserver (written in Go)

This application connects to teeworlds servers via its configured external console (econ).
It reads every logged line and checks for joining players.
The joining player's IP is then compared to the redis cache.
Does the cache not contain the IP, currently three VPN detection APIs are used to determine whether the player's IP is a VPN or not.
60% of these three APIs need to detect the IP as VPN in order for the application to actually ban the player and cache his VPN IP in the redis cache as such.

## Usage

If you have some predefined lists of VPN IPs, you may use `TeeworldsEconVPNDetection add` to add them to the redis database.
The same goes for whitelisted IPs with `TeeworldsEconVPNDetection remove`.
If multiple VPN detection APIs still decide to flag a player's ip, then there is no whitelisting that can save him from being banned.

You may use either docker to run the application by providing a `.env` file with the following value:
```dotenv
TWVPN_ECON_ADDRESSES=localhost:8404,localhost:8405
TWVPN_ECON_PASSWORDS="single password for all servers or comma separated list of passwords for each server"
TWVPN_REDIS_PASSWORD="your database password"

TWVPN_WHITELIST_TTL=168h30m30s
TWVPN_REDIS_DB_VPN=0 # 0-15
TWVPN_OFFLINE=false
TWVPN_IPHUB_TOKEN="N..."
TWVPN_PROXYCHECK_TOKEN="12345-1234-12345-123456"
TWVPN_VPNAPI_TOKEN="123456890abcdef"
TWVPN_PERMABAN_THRESHOLD="0.6"

TWVPN_VPN_BAN_REASON="VPN"
TWVPN_VPN_BAN_DURATION="24h30m30s"
```
And then start your containers using `make start` and stop them using `make stop`.

Alternatively, you may compile the application using the Go toolchain `go build .` and run the application by using more or less the exact same `.env` file as for docker and start the application with `./TeeworldsEconVPNDetection --config ./.env`.

You can also set up your redis database using docker with the provided `docker-compose.yml` file or just execute `make redis`.


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

## Command documentation
These are all of the available commands, subcommands and configuration parameters ofthe application.

### Run the econ log parser with the VPN detection.
```shell
$ ./TeeworldsEconVPNDetection --help
Environment variables:
  TWVPN_IPHUB_TOKEN           api key for https://iphub.info
  TWVPN_PROXYCHECK_TOKEN      api key for https://proxycheck.io
  TWVPN_VPNAPI_TOKEN          api key for https://vpnapi.io
  TWVPN_REDIS_ADDRESS          (default: "localhost:6379")
  TWVPN_REDIS_PASSWORD        optional password for the redis database
  TWVPN_REDIS_DB_VPN          redis database to use for the vpn ip data (0-15) (default: "15")
  TWVPN_NUTSDB_DIR            directory to store the nutsdb database (default: "./nutsdata")
  TWVPN_NUTSDB_BUCKET         bucket name for the nutsdb key value database (default: "whitelist")
  TWVPN_WHITELIST_TTL         time to live for whitelisted ips (default: "168h0m0s")
  TWVPN_ECON_ADDRESSES        comma separated list of econ addresses
  TWVPN_ECON_PASSWORDS        comma separated list of econ passwords
  TWVPN_RECONNECT_DELAY        (default: "10s")
  TWVPN_RECONNECT_TIMEOUT      (default: "24h0m0s")
  TWVPN_VPN_BAN_DURATION       (default: "5m0s")
  TWVPN_VPN_BAN_REASON         (default: "VPN")
  TWVPN_OFFLINE                if set to true no api calls will be made if an ip was not found in the database (= distributed ban server) (default: "false")
  TWVPN_PERMABAN_THRESHOLD    how many percent of the apis must agree on the vpn status for the IP to be added permanently to the blacklist (default: "0.6")
  TWVPN_IP_WHITELIST          comma separated list of files to whitelist
  TWVPN_IP_BLACKLIST          comma separated list of files to blacklist

Usage:
  TeeworldsEconVPNDetection [flags]
  TeeworldsEconVPNDetection [command]

Available Commands:
  add         add ips to the database (blacklist)
  completion  Generate completion script
  help        Help about any command
  remove      remove ips from the database (whitelist)

Flags:
  -c, --config string                .env config file path (or via env variable TWVPN_CONFIG)
      --econ-addresses string        comma separated list of econ addresses
      --econ-passwords string        comma separated list of econ passwords
  -h, --help                         help for TeeworldsEconVPNDetection
      --ip-blacklist string          comma separated list of files to blacklist
      --ip-whitelist string          comma separated list of files to whitelist
      --iphub-token string           api key for https://iphub.info
      --nutsdb-bucket string         bucket name for the nutsdb key value database (default "whitelist")
      --nutsdb-dir string            directory to store the nutsdb database (default "./nutsdata")
      --offline                       if set to true no api calls will be made if an ip was not found in the database (= distributed ban server)
      --permaban-threshold float     how many percent of the apis must agree on the vpn status for the IP to be added permanently to the blacklist (default 0.6)
      --proxycheck-token string      api key for https://proxycheck.io
      --reconnect-delay duration      (default 10s)
      --reconnect-timeout duration    (default 24h0m0s)
      --redis-address string          (default "localhost:6379")
      --redis-db-vpn int             redis database to use for the vpn ip data (0-15) (default 15)
      --redis-password string        optional password for the redis database
      --vpn-ban-duration duration     (default 5m0s)
      --vpn-ban-reason string         (default "VPN")
      --vpnapi-token string          api key for https://vpnapi.io
      --whitelist-ttl duration       time to live for whitelisted ips (default 168h0m0s)

Use "TeeworldsEconVPNDetection [command] --help" for more information about a command.
```

### Add ips to the database (blacklist)
```shell
$ ./TeeworldsEconVPNDetection add --help
Environment variables:
  TWVPN_REDIS_ADDRESS      (default: "localhost:6379")
  TWVPN_REDIS_PASSWORD
  TWVPN_REDIS_DB_VPN       (default: "15")

Usage:
  TeeworldsEconVPNDetection add blacklist.txt [more-banlists.txt...] [flags]

Flags:
  -c, --config string           .env config file path (or via env variable TWVPN_CONFIG)
  -h, --help                    help for add
      --redis-address string     (default "localhost:6379")
      --redis-db-vpn int         (default 15)
      --redis-password string
```

### Remove ips from the database (whitelist)
```shell
$ ./TeeworldsEconVPNDetection remove --help
Environment variables:
  TWVPN_REDIS_ADDRESS      (default: "localhost:6379")
  TWVPN_REDIS_PASSWORD
  TWVPN_REDIS_DB_VPN       (default: "15")

Usage:
  TeeworldsEconVPNDetection remove whitelist.txt [more-whitelists.txt...] [flags]

Flags:
  -c, --config string           .env config file path (or via env variable TWVPN_CONFIG)
  -h, --help                    help for remove
      --redis-address string     (default "localhost:6379")
      --redis-db-vpn int         (default 15)
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

### TODO:

Add proxy detection (if of a different currently running server)
```dotenv
PROXY_DETECTION_ENABLED=false
PROXY_UPDATE_INTERVAL=1m
PROXY_BAN_REASON="proxy connection"
PROXY_BAN_DURATION=24h
PROXY_SERVERNAME_DISTANCE=8
```