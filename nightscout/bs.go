package nightscout

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

func (n *NightscoutInstance) GetCurrentBloodSugar() (float64, error) {
	req, err := http.NewRequest("GET", n.bsUrl, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "sugarmonitor")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, errors.New("Non-200 response from Nightscout")
	}

	var nsResp NightscoutEntriesResponse

	err = json.NewDecoder(resp.Body).Decode(&nsResp)
	if err != nil {
		return 0, err
	}

	if nsResp.Status != 200 {
		return 0, fmt.Errorf("Non-200 status code from Nightscout: %d", nsResp.Status)
	}

	if len(nsResp.Result) == 0 {
		return 0, errors.New("No results returned from Nightscout")
	}

	nsResult := nsResp.Result[0]

	// example "sysTime": "2023-12-28T12:59:16.000Z",
	parsedNsTime, err := time.Parse(time.RFC3339, nsResult.SysTime)
	if err != nil {
		return 0, err
	}

	if time.Since(parsedNsTime) > 15*time.Minute {
		return 0, errors.New("Last reading not fresh, was more than 15 minutes ago")
	}

	mmol := float64(nsResult.Sgv) / 18.0

	return mmol, nil
}

type NightscoutEntriesResponse struct {
	Status int64    `json:"status"`
	Result []Result `json:"result"`
}

type Result struct {
	Sgv         int64  `json:"sgv"`
	Date        int64  `json:"date"`
	DateString  string `json:"dateString"`
	Trend       int64  `json:"trend"`
	Direction   string `json:"direction"`
	Device      string `json:"device"`
	Type        string `json:"type"`
	UTCOffset   int64  `json:"utcOffset"`
	SysTime     string `json:"sysTime"`
	Identifier  string `json:"identifier"`
	SrvModified int64  `json:"srvModified"`
	SrvCreated  int64  `json:"srvCreated"`
}
