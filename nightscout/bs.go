package nightscout

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type GlucoseReading struct {
	Identifier  string    `json:"id"`
	GlucoseMmol float64   `json:"glucose_mmol"`
	GlucoseMgdl float64   `json:"glucose_mgdl"`
	Time        time.Time `json:"timestamp"`
}

func (n *NightscoutInstance) GetCurrentBloodSugar() (*GlucoseReading, error) {
	req, err := http.NewRequest("GET", n.bsUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "sugarmonitor")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New("Non-200 response from Nightscout")
	}

	var nsResp NightscoutEntriesResponse

	err = json.NewDecoder(resp.Body).Decode(&nsResp)
	if err != nil {
		return nil, err
	}

	if nsResp.Status != 200 {
		return nil, fmt.Errorf("Non-200 status code from Nightscout: %d", nsResp.Status)
	}

	if len(nsResp.Result) == 0 {
		return nil, errors.New("No results returned from Nightscout")
	}

	nsResult := nsResp.Result[0]

	// example "sysTime": "2023-12-28T12:59:16.000Z",
	parsedNsTime, err := time.Parse(time.RFC3339, nsResult.SysTime)
	if err != nil {
		return nil, err
	}

	if time.Since(parsedNsTime) > 15*time.Minute {
		return nil, errors.New("Last reading not fresh, was more than 15 minutes ago")
	}

	mmol := float64(nsResult.Sgv) / 18.0

	gr := GlucoseReading{
		Identifier:  nsResult.Identifier,
		GlucoseMmol: mmol,
		Time:        parsedNsTime,
		GlucoseMgdl: float64(nsResult.Sgv),
	}

	return &gr, nil
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
