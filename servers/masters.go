package servers

import (
	"fmt"
	"log"
	"regexp"

	"github.com/go-resty/resty/v2"
	"github.com/jxsl13/TeeworldsEconVPNDetectionGo/config"
)

var (
	httpMasterUrl = config.New().HttpMasterServerUrl
	client        = (*resty.Client)(nil)
	path          = ""
)

func init() {
	hostSplitRegex := regexp.MustCompile(`^(https?://[^/]+)(/.*)$`)
	matches := hostSplitRegex.FindStringSubmatch(httpMasterUrl)
	if len(matches) != 3 {
		log.Fatalln("HttpMasterUrl is invalid")
	}
	// also split for custom urls
	host := matches[1]
	path = matches[2]

	client = resty.New().SetHostURL(host)
}

func GetServers() (*HttpMasterServerList, error) {
	serverList := &HttpMasterServerList{}
	resp, err := client.R().SetResult(serverList).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode()/100 != 2 {
		return nil, fmt.Errorf("failed to fetch http master server list: %s", resp.Status())
	}
	return serverList, nil
}

func GetHttpServerIPs() ([]string, error) {
	serverList, err := GetServers()
	if err != nil {
		return nil, err
	}
	return serverList.ServerIPs(), nil
}

func GetSimilarServers() (map[string][]Server, error) {
	h, err := GetServers()
	if err != nil {
		return nil, err
	}
	return h.SimilarServers(), nil
}
