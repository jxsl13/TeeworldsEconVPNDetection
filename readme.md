This application connects to teeworlds servers via its configured external console (econ).
It reads every logged line and checks for joining players.
The joining player's IP is then compared to the redis cache.
Does the cache not contain the IP, currently three VPN detection APIs are used to determine whether the player's IP is a VPN or not.
60% of these three APIs need to detect the IP as VPN in order for the application to actually ban the player and cache his VPN IP in the redis cache as such.

## Requirements

### Redis server for caching of IPs
This application requires a running redis database that can be used as cache for IPs.
The application caches non-VPN IPs in the redis database for one week.
VPN IPs are saved forever in order not to hit the free rate limit of the used APIs too fast.

#### Debian & Ubuntu
On Linux it is usually started automatically after its installation.

```
sudo apt install redis-server
```

#### macOS
On macOS you need to manually start it with `redis-server`.
```
brew install redis
```

## .env configuration file
The `.env` file contains the configuration.
Especially the econ addresses and password of the servers that this application should be attached to.

 - .env file in the same folder as the executable

#### Example .env
This file needs to live within the same directory as your executable.
```
# .env

# the api key can be found here: https://iphub.info/account (requires registration)
IPHUB_TOKEN=abcdefghijklmnopqrst0123456789

# use a proper email, it will most likely never be contacted, but in case you get a ban from GetIPIntel
EMAIL=john.doe@example.com

# econ addresses separated by one space
ECON_ADDRESSES=127.0.0.1:9303 127.0.0.1:9304 127.0.0.1:9305

# passwords, either one for all or one per address
ECON_PASSWORDS=abcdef0123456789

# each connection retries incrementally to reconnect to the server.
# if the connection timeout is reached, the routine stops.
RECONNECT_TIMEOUT_MINS=1440

# redis database credentials, these are the default values after installation
REDIS_ADDRESS=localhost:6379
REDIS_PASSWORD=

# how many minutes the VPN IP is banned and with what reason.
VPN_BANTIME=1440
VPN_BANREASON=VPN - discord.gg/ThGZ45e


```

## Downloading dependencies
```
go get -d
```

## Building
```
go build .
```

## Running
```
./TeeworldsEconVPNDetectionGo
```

## Add IPs from IPv4 text file
In order for this to work, you need to have a properly configured setup with a `.env` file.
Given a file with conents like:
```
1.236.132.203

# this adds/removes 2.56.0.1 through 2.56.255.254 to the database
2.56.92.0/16

# this adds/removes the IPs 2.56.140.1 through 2.56.140.254 from the database
2.56.140.0/24 
```
You can automatically add all of those IPs and the IPs from the subnets to your redis cache.
In order for such a file to be parsed, you can pass it on startup to the application like this:
```
# add IPs that are supposed to be banned to the database
./TeeworldsEconVPNDetectionGo -add ips.txt

# remove IPs from the database
./TeeworldsEconVPNDetectionGo -remove ips.txt 
```
You can use the `ip-v4.txt` from [VPNs](https://github.com/ejrv/VPNs).
It can take up to a few hours for all of those ~100 million IPs to be added to your redis database.

After all of the IPs have been parsed and added to the cache, the application shuts down.
You need to restart it without the flag in order to have the econ VPN detection behavior.

## Note
Currently no IPv6 support.


## Troubleshooting

##### EOF error
The Teeworlds server banned you for attempting to login too any times or for connecting too often.
Possible solution is to restart the game server. Should not be an issue, as the detector attempts to reconnect to the server.
