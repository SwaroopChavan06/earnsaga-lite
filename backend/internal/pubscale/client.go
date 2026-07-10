package pubscale

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const offersEndpoint = "https://api-ow.pubscale.com/v1/offer/api"

type Client struct {
	AppID  string
	PubKey string
	http   *http.Client
}

func NewClient(appID, pubKey string) *Client {
	return &Client{
		AppID:  appID,
		PubKey: pubKey,
		http:   &http.Client{Timeout: 15 * time.Second},
	}
}

type offersRequest struct {
	Page int `json:"page"`
	Size int `json:"size"`
}

// Raw response shapes matching PubScale's actual field names (short/abbreviated keys)
type OffersResponse struct {
	Offers []Offer `json:"offers"`
	Total  int     `json:"total"`
}

type Offer struct {
	ID       string    `json:"id"`
	UpdTS    int64     `json:"upd_ts"`
	OffType  string    `json:"off_type"`
	SID      string    `json:"s_id"`
	Name     string    `json:"name"`
	LpURL    string    `json:"lp_url"`
	Payout   Money     `json:"pyt"`
	InAppPay Money     `json:"inapp_pyt"`
	Creative Creative  `json:"crtvs"`
	Desc     Desc      `json:"desc"`
	Category []string  `json:"ctg"`
	Goals    []Goal    `json:"gls"`
	TrackURL string    `json:"trk_url"`
	OS       string    `json:"os"`
}

type Money struct {
	Currency string  `json:"cur"`
	Amount   float64 `json:"amt"`
}

type Creative struct {
	IconURL string `json:"ic_url"`
}

type Desc struct {
	Raw string `json:"raw"`
}

type Goal struct {
	ID           string  `json:"id"`
	Title        string  `json:"ttl"`
	Instructions string  `json:"instr"`
	Payout       Money   `json:"pyt"`
	Order        int     `json:"ord"`
}

// FetchOffers pulls one page of offers from the PubScale sandbox API.
// PubScale caches offers server-side and refreshes every 5 minutes, so
// there's no point polling more frequently than that during sync.
func (c *Client) FetchOffers(page, size int) (*OffersResponse, error) {
	body, err := json.Marshal(offersRequest{Page: page, Size: size})
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, offersEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("App-Id", c.AppID)
	req.Header.Set("Pub-Key", c.PubKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling pubscale offers api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("pubscale api returned status %d", resp.StatusCode)
	}

	var out OffersResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decoding pubscale response: %w", err)
	}

	return &out, nil
}
