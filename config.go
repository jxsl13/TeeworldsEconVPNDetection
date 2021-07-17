package main

import (
	"errors"
	"time"

	"github.com/go-redis/redis"
	configo "github.com/jxsl13/simple-configo"
	"github.com/jxsl13/simple-configo/parsers"
	"github.com/jxsl13/simple-configo/unparsers"
)

var (
	errRedisDatabaseNotFound   = errors.New("Could not connect to the redis database, check your REDIS_ADDRESS, REDIS_PASSWORD and make sure your redis database is running")
	errAddressPasswordMismatch = errors.New("The number of ECON_PASSWORD doesn't match the number of ECON_ADDRESSES, either provide one password for all addresses or one password per address")
)

// Config represents the application configuration
type Config struct {
	IPHubToken       string
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
	ZCatchLogFormat  bool

	// hidden
	delimiter string
}

func (c *Config) Name() string {
	return "TeeworldsEconVPNDetectionGo"
}

func (c *Config) Options() configo.Options {

	return configo.Options{
		{
			Key:             "DELIMITER",
			Description:     "delimiting character that is used to split lists",
			DefaultValue:    " ",
			ParseFunction:   parsers.String(&c.delimiter),
			UnparseFunction: unparsers.String(&c.delimiter),
		},
		{
			Key:             "IPHUB_TOKEN",
			Description:     "API key that is provided via any tier by https://iphub.info",
			ParseFunction:   parsers.String(&c.IPHubToken),
			UnparseFunction: unparsers.String(&c.IPHubToken),
		},
		{
			Key:             "REDIS_ADDRESS",
			Mandatory:       true,
			Description:     "address of your redis database",
			DefaultValue:    "localhost:6379",
			ParseFunction:   parsers.String(&c.RedisAddress),
			UnparseFunction: unparsers.String(&c.RedisAddress),
		},
		{
			Key:             "REDIS_DB_VPN",
			Mandatory:       true,
			Description:     "database to use in your redis instance [0,15](default: 0)",
			DefaultValue:    "1",
			ParseFunction:   parsers.ChoiceInt(&c.RedisDB, 0, 15),
			UnparseFunction: unparsers.Int(&c.RedisDB),
		},
		{
			Key:             "ECON_ADDRESSES",
			Mandatory:       true,
			Description:     "a single space separated list of Teeworlds econ addresses",
			ParseFunction:   parsers.List(&c.EconServers, &c.delimiter),
			UnparseFunction: unparsers.List(&c.EconServers, &c.delimiter),
		},
		{
			Key:             "ECON_PASSWORDS",
			Mandatory:       true,
			Description:     "a single space separated list of Teeworlds econ passwords. enter a single password for all servers.",
			ParseFunction:   parsers.List(&c.EconPasswords, &c.delimiter),
			UnparseFunction: unparsers.List(&c.EconPasswords, &c.delimiter),
		},
		{
			Key:             "RECONNECT_TIMEOUT",
			DefaultValue:    "24h",
			Description:     "after how much time a reconnect is concidered not feasible (default: 5m, may use 1h5m10s500ms)",
			ParseFunction:   parsers.Duration(&c.ReconnectTimeout),
			UnparseFunction: unparsers.Duration(&c.ReconnectTimeout),
		},
		{
			Key:             "RECONNECT_DELAY",
			DefaultValue:    "10s",
			Description:     "after how much time to try reconnecting to the database again.",
			ParseFunction:   parsers.Duration(&c.ReconnectDelay),
			UnparseFunction: unparsers.Duration(&c.ReconnectDelay),
		},
		{
			Key:             "VPN_BANREASON",
			DefaultValue:    "VPN",
			Description:     "ban reason message",
			ParseFunction:   parsers.String(&c.VPNBanReason),
			UnparseFunction: unparsers.String(&c.VPNBanReason),
		},
		{
			Key:             "VPN_BANTIME",
			DefaultValue:    "5m",
			Description:     "for how long a vpn client is banned.",
			ParseFunction:   parsers.Duration(&c.VPNBanTime),
			UnparseFunction: unparsers.Duration(&c.VPNBanTime),
		},
		{
			Key:             "ZCATCH_LOGGING",
			DefaultValue:    "false",
			Description:     "whether to use the zCatch log format parser or the vanilla parser",
			ParseFunction:   parsers.Bool(&c.ZCatchLogFormat),
			UnparseFunction: unparsers.Bool(&c.ZCatchLogFormat),
		},
		{
			Key:             "OFFLINE",
			DefaultValue:    "false",
			Description:     "offline solely uses the redis database to evaluate the ips. no online services are used.",
			ParseFunction:   parsers.Bool(&c.Offline),
			UnparseFunction: unparsers.Bool(&c.Offline),
		},
	}
}

// NewConfig creates a new configuration file based on
// the data that has been retrieved from the .env environment file.
func NewConfig(dotEnvFilePath string) (*Config, error) {
	cfg := &Config{}

	err := configo.ParseEnvFile(dotEnvFilePath, cfg)
	if err != nil {
		return cfg, err
	}

	options := redis.Options{
		Addr:     cfg.RedisAddress,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}

	redisClient := redis.NewClient(&options)
	defer redisClient.Close()

	pong, err := redisClient.Ping().Result()
	if err != nil || pong != "PONG" {
		return cfg, errRedisDatabaseNotFound
	}

	// add password for every econ server.
	if len(cfg.EconServers) != len(cfg.EconPasswords) {
		if len(cfg.EconServers) > 1 && len(cfg.EconPasswords) > 1 {
			return cfg, errAddressPasswordMismatch
		}
		if len(cfg.EconServers) > 1 && len(cfg.EconPasswords) == 1 {
			for len(cfg.EconPasswords) < len(cfg.EconServers) {
				cfg.EconPasswords = append(cfg.EconPasswords, cfg.EconPasswords[0])
			}
		}
	}
	return cfg, nil

}
