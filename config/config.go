package config

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-redis/redis"
	"github.com/jxsl13/TeeworldsEconVPNDetectionGo/vpn"
	"github.com/jxsl13/goripr"
	configo "github.com/jxsl13/simple-configo"
	"github.com/jxsl13/simple-configo/actions"
	"github.com/jxsl13/simple-configo/parsers"
	"github.com/jxsl13/simple-configo/unparsers"
)

var (
	errRedisDatabaseNotFound   = errors.New("could not connect to the redis database, check your REDIS_ADDRESS, REDIS_PASSWORD and make sure your redis database is running")
	errAddressPasswordMismatch = errors.New("the number of ECON_PASSWORD doesn't match the number of ECON_ADDRESSES, either provide one password for all addresses or one password per address")

	cfg       = (*Config)(nil)
	once      sync.Once
	closeOnce sync.Once

	// DefaultEnvFile can be changed to a different env file location.
	DefaultEnvFile = ".env"
)

// New creates a new configuration file based on
// the data that has been retrieved from the .env environment file.
// any call after the first one will return the config of the first call
// the location of the .env file can be changed via the DefaultEnvFile variable
func New() *Config {
	return newFromFile(DefaultEnvFile)
}

func newFromFile(dotEnvFilePath string) *Config {
	if cfg != nil {
		return cfg
	}

	once.Do(func() {
		c := &Config{}
		err := configo.ParseEnvFile(dotEnvFilePath, c)
		if err != nil {
			log.Fatalln(err)
		}

		cfg = c
	})
	return cfg
}

// Config represents the application configuration
type Config struct {
	IPHubToken      string
	ProxyCheckToken string
	IpTeohEnabled   bool

	RedisAddress     string
	RedisPassword    string
	RedisDB          int
	EconServers      []string
	EconPasswords    []string
	ReconnectDelay   time.Duration
	ReconnectTimeout time.Duration
	VPNBanTime       time.Duration
	VPNBanReason     string
	Offline          bool
	ZCatchLogFormat  bool // parsing of zCatch econ logs (Vanilla log format -> false)

	ProxyUpdateInterval time.Duration
	ProxyBanDuration    time.Duration
	ProxyBanReason      string

	AddFile    *string // add ip list to cache
	RemoveFile *string // remove ip list from cache (executed after adding)

	ripr              *goripr.Client // redis cache client
	checker           *VPNChecker    // online api checker client
	PermaBanThreshold float64        // how many percent of the checker apis needed to perma ban the checked ip

	ctx    context.Context    // central context that is canceled once the config is closed
	cancel context.CancelFunc // called on unparse
}

func (c *Config) Options() configo.Options {

	delimiter := ""

	return configo.Options{
		{
			Key:           "DELIMITER",
			Description:   "delimiting character that is used to split lists",
			DefaultValue:  " ",
			ParseFunction: parsers.String(&delimiter),
		},
		{
			Key:           "IPHUB_TOKEN",
			Description:   "API key that is provided via any tier by https://iphub.info",
			ParseFunction: parsers.String(&c.IPHubToken),
		},
		{
			Key:           "PROXYCHECK_TOKEN",
			Description:   "API key that is provided via https://proxycheck.io",
			ParseFunction: parsers.String(&c.ProxyCheckToken),
		},
		{
			Key:           "IPTEOH_ENABLED",
			Description:   "Wether to use the https://ip.teoh.io api",
			DefaultValue:  "false",
			ParseFunction: parsers.Bool(&c.IpTeohEnabled),
		},
		{
			Key:           "REDIS_ADDRESS",
			Mandatory:     true,
			Description:   "address of your redis database",
			DefaultValue:  "localhost:6379",
			ParseFunction: parsers.String(&c.RedisAddress),
		},
		{
			Key:           "REDIS_PASSWORD",
			Description:   "password of your redis database",
			ParseFunction: parsers.String(&c.RedisPassword),
		},
		{
			Key:           "REDIS_DB_VPN",
			Mandatory:     true,
			Description:   "database to use in your redis instance [0,15](default: 0)",
			DefaultValue:  "0",
			ParseFunction: parsers.RangesInt(&c.RedisDB, 0, 15),
		},
		{
			Key: "Redis Ping Pong Check",
			PreParseAction: func() error {
				options := redis.Options{
					Addr:     c.RedisAddress,
					Password: c.RedisPassword,
					DB:       c.RedisDB,
				}

				redisClient := redis.NewClient(&options)
				defer redisClient.Close()

				pong, err := redisClient.Ping().Result()
				if err != nil || pong != "PONG" {
					return fmt.Errorf("%w: %v", errRedisDatabaseNotFound, err)
				}
				return nil
			},
		},
		{
			Key:             "ECON_ADDRESSES",
			Mandatory:       true,
			Description:     "a single space separated list of Teeworlds econ addresses",
			ParseFunction:   parsers.List(&c.EconServers, &delimiter),
			UnparseFunction: unparsers.List(&c.EconServers, &delimiter),
		},
		{
			Key:             "ECON_PASSWORDS",
			Mandatory:       true,
			Description:     "a single space separated list of Teeworlds econ passwords. enter a single password for all servers.",
			ParseFunction:   parsers.List(&c.EconPasswords, &delimiter),
			UnparseFunction: unparsers.List(&c.EconPasswords, &delimiter),
		},
		{
			Key: "ECON_ADDRESSES & ECON_PASSWORDS consolidation",
			PreParseAction: func() error {
				// add password for every econ server.
				if len(c.EconServers) != len(c.EconPasswords) {
					if len(c.EconServers) > 1 && len(c.EconPasswords) > 1 {
						return errAddressPasswordMismatch
					}
					if len(c.EconServers) > 1 && len(c.EconPasswords) == 1 {
						for len(c.EconPasswords) < len(c.EconServers) {
							c.EconPasswords = append(c.EconPasswords, c.EconPasswords[0])
						}
					}
				}
				return nil
			},
		},
		{
			Key:           "RECONNECT_TIMEOUT",
			DefaultValue:  "24h",
			Description:   "after how much time a reconnect is concidered not feasible (default: 5m, may use 1h5m10s500ms)",
			ParseFunction: parsers.Duration(&c.ReconnectTimeout),
		},
		{
			Key:           "RECONNECT_DELAY",
			DefaultValue:  "10s",
			Description:   "after how much time to try reconnecting to the database again.",
			ParseFunction: parsers.Duration(&c.ReconnectDelay),
		},
		{
			Key:           "VPN_BAN_REASON",
			DefaultValue:  "VPN",
			Description:   "ban reason message",
			ParseFunction: parsers.String(&c.VPNBanReason),
		},
		{
			Key:           "VPN_BAN_TIME",
			DefaultValue:  "5m",
			Description:   "for how long a vpn client is banned.",
			ParseFunction: parsers.Duration(&c.VPNBanTime),
		},
		{
			Key:           "ZCATCH_LOGGING",
			DefaultValue:  "false",
			Description:   "whether to use the zCatch log format parser or the vanilla parser",
			ParseFunction: parsers.Bool(&c.ZCatchLogFormat),
		},
		{
			Key:           "PROXY_UPDATE_INTERVAL",
			DefaultValue:  "5m",
			Description:   "how long to wait before updating the IP list of registered Teeworlds servers that might act as proxies",
			ParseFunction: parsers.Duration(&c.ProxyUpdateInterval),
		},
		{
			Key:           "PROXY_BAN_REASON",
			DefaultValue:  "proxy connection",
			Description:   "The ban reason for users that connect through a proxy game server (stealing accounts etc)",
			ParseFunction: parsers.String(&c.ProxyBanReason),
		},
		{
			Key:           "PROXY_BAN_DURATION",
			DefaultValue:  "24h",
			Description:   "How long to ban proxy servers",
			ParseFunction: parsers.Duration(&c.ProxyBanDuration),
		},
		{
			Key:           "OFFLINE",
			DefaultValue:  "false",
			Description:   "offline solely uses the redis database to evaluate the ips. no online services are used.",
			ParseFunction: parsers.Bool(&c.Offline),
		},
		{
			Key: "Parse Flags",
			PreParseAction: func() error {
				addFile := ""
				removeFile := ""
				offline := false
				flag.StringVar(&addFile, "add", "", "pass a text file with IPs and IP subnets to be added to the database")
				flag.StringVar(&removeFile, "remove", "", "pass a text file with IPs and IP subnets to be removed from the database")
				flag.BoolVar(&offline, "offline", false, "do not use the api endpoints, only rely on the cache")
				flag.Parse()

				if addFile != "" {
					c.AddFile = &addFile
				} else if removeFile != "" {
					c.RemoveFile = &removeFile
				}

				// only overwrite if flag is set to true
				if !c.Offline && offline {
					c.Offline = offline
				}

				return nil
			},
		},
		{
			Key: "Add IPs to Redis Cache & Remove IPs from Redis Cache",
			PreParseAction: actions.OnlyIf(c.AddFile != nil, func() error {
				_, err := parseFileAndAddIPsToCache(*c.AddFile, c.RedisAddress, c.RedisPassword, c.RedisDB)
				return err
			}),
			PostParseAction: actions.OnlyIf(c.RemoveFile != nil, func() error {
				_, err := parseFileAndRemoveIPsFromCache(*c.RemoveFile, c.RedisAddress, c.RedisPassword, c.RedisDB)
				return err
			}),
		},
		{
			Key: "Initialize Goripr",
			PreParseAction: func() error {
				ripr, err := goripr.NewClient(goripr.Options{
					Addr:     c.RedisAddress,
					Password: c.RedisPassword,
					DB:       c.RedisDB,
				})
				if err != nil {
					return err
				}
				c.ripr = ripr
				return nil
			},
			PreUnparseAction: func() error {
				// called on unparsing
				return c.ripr.Close()
			},
		},
		{
			Key: "Initialize Online VPN Checker",
			PreParseAction: func() error {
				c.checker = newVPNChecker(c)
				return nil
			},
			PreUnparseAction: func() error {
				// called on unparsing
				return c.checker.Close()
			},
		},
		{
			Key:           "PERMABAN_THRESHOLD",
			Description:   "How many percent of all of the vpn detection apis need to detect an ip as VPN in order for it to be permanently banned",
			DefaultValue:  "0.6",
			ParseFunction: parsers.Float(&c.PermaBanThreshold, 64),
		},
		{
			Key: "Initialize Context",
			PreParseAction: func() error {
				c.ctx, c.cancel = context.WithCancel(context.Background())
				return nil
			},
			PreUnparseAction: func() error {
				// called on close
				c.cancel()
				return nil
			},
		},
	}
}

// apis returns a list of available apis that is constructed based on the configuration
func (c *Config) apis() []vpn.VPN {
	apis := []vpn.VPN{}
	if !c.Offline {
		// share client with all apis
		httpClient := &http.Client{}

		if cfg.IPHubToken != "" {
			apis = append(apis, vpn.NewIPHub(httpClient, cfg.IPHubToken))
		}

		if cfg.IpTeohEnabled {
			apis = append(apis, vpn.NewIPTeohIO(httpClient))
		}

		if cfg.ProxyCheckToken != "" {
			apis = append(apis, vpn.NewProxyCheck(httpClient, cfg.ProxyCheckToken))
		}
	}
	return apis
}

func (c *Config) Context() context.Context {
	return c.ctx
}

func (c *Config) Checker() *VPNChecker {
	return c.checker
}

func (c *Config) UpdateIPsTicker() *time.Ticker {
	return time.NewTicker(c.ProxyUpdateInterval)
}

// Close should be called in your main function with a defer
func Close() {
	closeOnce.Do(func() {
		_, err := configo.Unparse(cfg)
		if err != nil {
			log.Println(err)
		}
	})
}
