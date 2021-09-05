package servers

import (
	"fmt"
	"log"
	"regexp"

	"github.com/go-resty/resty/v2"
)

var (
	HttpMasterUrl = "https://master1.ddnet.tw/ddnet/15/servers.json"
	client        = (*resty.Client)(nil)
	path          = ""
)

func init() {
	hostSplitRegex := regexp.MustCompile(`^(https?://[^/]+)(/.*)$`)
	matches := hostSplitRegex.FindStringSubmatch(HttpMasterUrl)
	if len(matches) != 3 {
		log.Fatalln("HttpMasterUrl is invalid")
	}
	// also split for custom urls
	host := matches[1]
	path = matches[2]

	client = resty.New().SetHostURL(host)
}

func GetHttpServerIPs() ([]string, error) {
	serverList := &HttpMasterServerList{}
	resp, err := client.R().SetResult(serverList).Get(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode()/100 != 2 {
		return nil, fmt.Errorf("failed to fetch http master server list: %s", resp.Status())
	}
	return serverList.ServerIPs(), nil
}
