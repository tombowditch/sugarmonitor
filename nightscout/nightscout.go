package nightscout

import (
	"errors"
	"net/url"
	"os"
)

var errNoUrl = errors.New("NIGHTSCOUT_URL environment variable not set")

type NightscoutInstance struct {
	bsUrl string
}

func NewNightscout() (*NightscoutInstance, error) {
	rawUrl := os.Getenv("NIGHTSCOUT_URL")
	if rawUrl == "" {
		return nil, errNoUrl
	}

	nsUrl, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}

	token := os.Getenv("NIGHTSCOUT_TOKEN")

	nsUrl.Path = "/api/v3/entries"

	q := nsUrl.Query()
	if token != "" {
		q.Set("token", token)
	}
	q.Set("limit", "1")
	q.Set("sort$desc", "date")

	nsUrl.RawQuery = q.Encode()

	return &NightscoutInstance{
		bsUrl: nsUrl.String(),
	}, nil
}
