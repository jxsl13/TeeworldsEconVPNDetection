package config

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jxsl13/TeeworldsEconVPNDetectionGo/vpn"
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
	return &Config{
		IPTeohEnabled:    false,
		RedisAddress:     "localhost:6379",
		RedisDB:          0,
		ReconnectDelay:   10 * time.Second,
		ReconnectTimeout: 24 * time.Hour,
		VPNBanReason:     "VPN",
		VPNBanTime:       5 * time.Minute,
		BanThreshold:     0.6,
	}
}

// Config represents the application configuration
type Config struct {
	IPHubToken      string `koanf:"iphub.token"`
	ProxyCheckToken string `koanf:"proxycheck.token"`
	IPTeohEnabled   bool   `koanf:"ipteoh.enabled"`

	RedisAddress  string `koanf:"redis.address" validate:"required"`
	RedisPassword string `koanf:"redis.password" validate:"required"`
	RedisDB       int    `koanf:"redis.db.vpn"`

	EconServersString string `koanf:"econ.addresses" validate:"required" description:"comma separated list of econ addresses"`
	EconServers       []string

	EconPasswordsString string `koanf:"econ.passwords" validate:"required" description:"comma separated list of econ passwords"`
	EconPasswords       []string
	ReconnectDelay      time.Duration `koanf:"reconnect.delay" validate:"required"`
	ReconnectTimeout    time.Duration `koanf:"reconnect.timeout" validate:"required"`
	VPNBanTime          time.Duration `koanf:"vpn.ban.duration" validate:"required"`
	VPNBanReason        string        `koanf:"vpn.ban.reason" validate:"required"`
	Offline             bool          `koanf:"offline"`

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

	return nil
}

// apis returns a list of available apis that is constructed based on the configuration
func (c *Config) APIs() []vpn.VPN {
	apis := []vpn.VPN{}
	if !c.Offline {
		// share client with all apis
		httpClient := &http.Client{}

		if c.IPHubToken != "" {
			apis = append(apis, vpn.NewIPHub(httpClient, c.IPHubToken))
		}

		if c.IPTeohEnabled {
			apis = append(apis, vpn.NewIPTeohIO(httpClient))
		}

		if c.ProxyCheckToken != "" {
			apis = append(apis, vpn.NewProxyCheck(httpClient, c.ProxyCheckToken))
		}
	}
	return apis
}
