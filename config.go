package main

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
)

type address string  // ip:port
type token string    // long weird string
type password string // password string

var (
	errIPHubTokenMissing       = errors.New("The IPHub api access key is missing, IPHUB_TOKEN")
	errRedisDatabaseNotFound   = errors.New("could not connect to the redis database, check your REDIS_ADDRESS, REDIS_PASSWORD and make sure your redis database is running")
	errEconAddressesMissing    = errors.New("please provide some econ addresses in your .env configuration: 'ECON_LIST=127.0.0.1:1234 127.0.0.1:5678'")
	errAddressPasswordMismatch = errors.New("the number of ECON_PASSWORD doesn't match the number of ECON_ADDRESSES, either provide one password for all addresses or one password per address")
)

// Config represents the application configuration
type Config struct {
	IPHubToken       token
	RedisAddress     address
	RedisPassword    password
	EconServers      []address
	EconPasswords    []password
	ReconnectTimeout time.Duration
	VPNBanTime       time.Duration
	VPNBanReason     string
	Offline          bool
}

// NewConfig creates a new configuration file based on
// the data that has been retrieved from the .env environment file.
func NewConfig(env map[string]string) (Config, error) {
	cfg := Config{}

	// retrieved from .env file
	IPHubToken := env["IPHUB_TOKEN"]

	if len(IPHubToken) == 0 {
		return cfg, errIPHubTokenMissing
	}
	cfg.IPHubToken = token(IPHubToken)

	RedisAddress := env["REDIS_ADDRESS"]
	if RedisAddress == "" {
		RedisAddress = "localhost:6379"
	}

	RedisPassword := env["REDIS_PASSWORD"]

	options := redis.Options{
		Addr:     RedisAddress,
		Password: RedisPassword,
	}

	redisClient := redis.NewClient(&options)
	defer redisClient.Close()

	pong, err := redisClient.Ping().Result()
	if err != nil || pong != "PONG" {
		return cfg, errRedisDatabaseNotFound
	}

	cfg.RedisAddress = address(RedisAddress)
	cfg.RedisPassword = password(RedisPassword)

	EconAddresses := strings.Split(env["ECON_ADDRESSES"], " ")
	if len(EconAddresses) == 0 {
		return cfg, errEconAddressesMissing
	}

	cfg.EconServers = make([]address, 0, len(EconAddresses))
	for _, addr := range EconAddresses {
		cfg.EconServers = append(cfg.EconServers, address(addr))
	}

	EconPasswords := strings.Split(env["ECON_PASSWORDS"], " ")
	if len(EconAddresses) == 0 || len(EconPasswords) == 0 {
		return cfg, errAddressPasswordMismatch
	}

	if len(EconAddresses) != len(EconPasswords) {
		if len(EconAddresses) > 1 && len(EconPasswords) > 1 {
			return cfg, errAddressPasswordMismatch
		}
		if len(EconAddresses) > 1 && len(EconPasswords) == 1 {
			for len(EconPasswords) < len(EconAddresses) {
				EconPasswords = append(EconPasswords, EconPasswords[0])
			}
		}
	}

	cfg.EconPasswords = make([]password, 0, len(EconPasswords))
	for _, pw := range EconPasswords {
		cfg.EconPasswords = append(cfg.EconPasswords, password(pw))
	}

	ReconnectTimeoutMinutes, err := strconv.Atoi(env["RECONNECT_TIMEOUT_MINS"])
	if err != nil || ReconnectTimeoutMinutes <= 0 {
		ReconnectTimeoutMinutes = 60
	}
	cfg.ReconnectTimeout = time.Minute * time.Duration(ReconnectTimeoutMinutes)

	cfg.VPNBanReason = env["VPN_BANREASON"]

	bantime, err := strconv.Atoi(env["VPN_BANTIME"])
	if err != nil {
		bantime = 5
	}
	cfg.VPNBanTime = time.Duration(bantime) * time.Minute

	return cfg, nil

}
