package main

import (
	"errors"
	"log"
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
	errRedisDatabaseNotFound   = errors.New("Could not connect to the redis database, check your REDIS_ADDRESS, REDIS_PASSWORD and make sure your redis database is running")
	errEconAddressesMissing    = errors.New("Please provide some econ addresses in your .env configuration: 'ECON_LIST=127.0.0.1:1234 127.0.0.1:5678'")
	errAddressPasswordMismatch = errors.New("The number of ECON_PASSWORD doesn't match the number of ECON_ADDRESSES, either provide one password for all addresses or one password per address")
	errNoVPNBanReasonSpecified = errors.New("Please provide a non-empty VPN_BANREASON that is used as ban reason")
)

// Config represents the application configuration
type Config struct {
	IPHubToken       token
	RedisAddress     address
	RedisPassword    password
	RedisDB          int
	EconServers      []address
	EconPasswords    []password
	ReconnectDelay   time.Duration
	ReconnectTimeout time.Duration
	VPNBanTime       time.Duration
	VPNBanReason     string
	Offline          bool
	zCatchLogFormat  bool
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
		log.Println("Using default REDIS_ADDRESS localhost:6379")
		RedisAddress = "localhost:6379"
	}

	RedisPassword := env["REDIS_PASSWORD"]

	RedisDBStr := env["REDIS_DB_VPN"]
	if RedisDBStr == "" {
		RedisDBStr = "0"
	}

	RedisDB, err := strconv.Atoi(RedisDBStr)
	if err != nil {
		RedisDB = 0
		log.Println("Using redis database:", RedisDB)
	}

	options := redis.Options{
		Addr:     RedisAddress,
		Password: RedisPassword,
		DB:       RedisDB,
	}

	redisClient := redis.NewClient(&options)
	defer redisClient.Close()

	pong, err := redisClient.Ping().Result()
	if err != nil || pong != "PONG" {
		return cfg, errRedisDatabaseNotFound
	}

	cfg.RedisAddress = address(RedisAddress)
	cfg.RedisPassword = password(RedisPassword)
	cfg.RedisDB = RedisDB

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
		log.Println("Using default RECONNECT_TIMEOUT_MINS of 5 (minutes)")
		ReconnectTimeoutMinutes = 5
	}
	cfg.ReconnectTimeout = time.Minute * time.Duration(ReconnectTimeoutMinutes)

	ReconnectDelaySeconds, err := strconv.Atoi(env["RECONNECT_DELAY_SECONDS"])
	if err != nil || ReconnectDelaySeconds <= 0 {
		log.Println("Using default RECONNECT_DELAY_SECONDS of 10 (seconds)")
		ReconnectTimeoutMinutes = 10
	}
	cfg.ReconnectDelay = time.Second * time.Duration(ReconnectDelaySeconds)

	banReason, ok := env["VPN_BANREASON"]
	if !ok {
		return cfg, errNoVPNBanReasonSpecified
	}
	cfg.VPNBanReason = banReason
	log.Println("General VPN ban reason(VPN_BANREASON):", cfg.VPNBanReason)

	bantime, err := strconv.Atoi(env["VPN_BANTIME"])
	if err != nil {
		log.Println("Using default VPN_BANTIME of 5 (minutes)")
		bantime = 5
	}
	cfg.VPNBanTime = time.Duration(bantime) * time.Minute

	zCatchLogging := env["ZCATCH_LOGGING"]
	switch zCatchLogging {
	case "1", "true", "enable", "enabled", "on":
		log.Println("Using zCatch log parsing.")
		cfg.zCatchLogFormat = true
	default:
		log.Println("Using Teeworlds Vanilla log parsing.")
		cfg.zCatchLogFormat = false
	}

	return cfg, nil

}
