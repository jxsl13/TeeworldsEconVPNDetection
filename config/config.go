package config

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jxsl13/TeeworldsEconVPNDetection/vpn"
	"github.com/redis/go-redis/v9"
)

var (
	errRedisDatabaseNotFound   = errors.New("could not connect to the redis database, check your REDIS_ADDRESS, REDIS_PASSWORD and make sure your redis database is running")
	errAddressPasswordMismatch = errors.New("the number of ECON_PASSWORD doesn't match the number of ECON_ADDRESSES, either provide one password for all addresses or one password per address")
)

// New creates a new configuration file based on
// the data that has been retrieved from the .env environment file.
// any call after the first one will return the config of the first call
// the location of the .env file can be changed via the DefaultEnvFile variable
func New() *Config {
	pwd := "./"
	dir, err := os.Getwd()
	if err == nil {
		pwd = dir
	}
	nutsDir := filepath.Join(pwd, "nutsdata")

	return &Config{
		RedisAddress: "localhost:6379",
		RedisDB:      15,
		NutsDBDir:    nutsDir,
		NutsDBBucket: "whitelist",
		WhitelistTTL: 7 * 24 * time.Hour,

		ReconnectDelay:   10 * time.Second,
		ReconnectTimeout: 24 * time.Hour,
		VPNBanReason:     "VPN",
		VPNBanTime:       5 * time.Minute,
		BanThreshold:     0.6,
	}
}

// Config represents the application configuration
type Config struct {
	IPHubToken      string `koanf:"iphub.token" description:"api key for https://iphub.info"`
	ProxyCheckToken string `koanf:"proxycheck.token" description:"api key for https://proxycheck.io"`
	VPNApiToken     string `koanf:"vpnapi.token" description:"api key for https://vpnapi.io"`

	RedisAddress  string `koanf:"redis.address" validate:"required"`
	RedisPassword string `koanf:"redis.password" description:"optional password for the redis database"`
	RedisDB       int    `koanf:"redis.db.vpn" validate:"gte=0,lte=15" description:"redis database to use for the vpn ip data (0-15)"`

	NutsDBDir    string        `koanf:"nutsdb.dir" validate:"required" description:"directory to store the nutsdb database"`
	NutsDBBucket string        `koanf:"nutsdb.bucket" validate:"required" description:"bucket name for the nutsdb key value database"`
	WhitelistTTL time.Duration `koanf:"whitelist.ttl" validate:"required" description:"time to live for whitelisted ips"`

	EconServersString string `koanf:"econ.addresses" validate:"required" description:"comma separated list of econ addresses"`
	EconServers       []string

	EconPasswordsString string `koanf:"econ.passwords" validate:"required" description:"comma separated list of econ passwords"`
	EconPasswords       []string
	ReconnectDelay      time.Duration `koanf:"reconnect.delay" validate:"required"`
	ReconnectTimeout    time.Duration `koanf:"reconnect.timeout" validate:"required"`
	VPNBanTime          time.Duration `koanf:"vpn.ban.duration" validate:"required"`
	VPNBanReason        string        `koanf:"vpn.ban.reason" validate:"required"`
	Offline             bool          `koanf:"offline" description:" if set to true no api calls will be made if an ip was not found in the database (= distributed ban server)"`

	BanThreshold float64 `koanf:"perma.ban.threshold" validate:"required"`

	Whitelist string `koanf:"ip.whitelist" description:"comma separated list of ip ranges to whitelist"`
	Blacklist string `koanf:"ip.blacklist" description:"comma separated list of ip ranges to blacklist"`

	Whitelists []string
	Blacklists []string
}

func (c *Config) Validate() error {
	err := validator.New().Struct(c)
	if err != nil {
		return err
	}

	c.EconServers = strings.Split(c.EconServersString, ",")
	c.EconPasswords = strings.Split(c.EconPasswordsString, ",")
	c.Whitelists = strings.Split(c.Whitelist, ",")
	c.Blacklists = strings.Split(c.Blacklist, ",")

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

	options := redis.Options{
		Addr:     c.RedisAddress,
		Password: c.RedisPassword,
		DB:       c.RedisDB,
	}

	redisClient := redis.NewClient(&options)
	defer redisClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pong, err := redisClient.Ping(ctx).Result()
	if err != nil || pong != "PONG" {
		return fmt.Errorf("%w: %v", errRedisDatabaseNotFound, err)
	}

	if c.WhitelistTTL < time.Second {
		return errors.New("whitelist ttl must be at least 1 second")
	}

	return nil
}

// apis returns a list of available apis that is constructed based on the configuration
func (c *Config) APIs() []vpn.VPN {
	apis := []vpn.VPN{}
	if !c.Offline {
		// share client with all apis
		// client reuses tls connections
		httpClient := &http.Client{}

		if c.IPHubToken != "" {
			apis = append(apis, vpn.NewIPHub(httpClient, c.IPHubToken))
		}

		if c.VPNApiToken != "" {
			apis = append(apis, vpn.NewVPNAPI(httpClient, c.VPNApiToken))
		}

		if c.ProxyCheckToken != "" {
			apis = append(apis, vpn.NewProxyCheck(httpClient, c.ProxyCheckToken))
		}
	}
	return apis
}
