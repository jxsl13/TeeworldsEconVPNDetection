module github.com/jxsl13/TeeworldsEconVPNDetectionGo

require (
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/go-resty/resty/v2 v2.6.0
	github.com/jxsl13/goripr v1.1.1
	github.com/jxsl13/simple-configo v1.21.1
	github.com/jxsl13/twapi v1.3.1
	github.com/onsi/gomega v1.16.0 // indirect
)

replace (
	github.com/jxsl13/TeeworldsEconVPNDetectionGo/config => ./config
	github.com/jxsl13/TeeworldsEconVPNDetectionGo/econ => ./econ
	github.com/jxsl13/TeeworldsEconVPNDetectionGo/vpn => ./vpn
)

go 1.13
