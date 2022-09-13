package infra

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/devsagul/gophemart/internal/core"
	"github.com/devsagul/gophemart/internal/storage"
	"github.com/shopspring/decimal"
)

type orderResponse struct {
	Order   string           `json:"order"`
	Status  string           `json:"status"`
	Accrual *decimal.Decimal `json:"accrual"`
}

// worker co-routine to process orders from given channel
func Worker(
	orders <-chan *core.Order,
	apiAddress string,
	store storage.Storage,
) error {
	originalAPIURL, err := url.Parse(apiAddress)

	if err != nil {
		return err
	}

	for order := range orders {
		apiURL := *originalAPIURL

		apiURL.Path = path.Join(originalAPIURL.Path, fmt.Sprintf("/api/orders/%s", order.ID))
		resp, err := http.Get(apiURL.String())
		if err != nil {
			log.Printf("Error during orders processing: %v", err)
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error while reading accrual system's response: %v", err)
			continue
		}

		err = resp.Body.Close()
		if err != nil {
			log.Printf("Could not close response body: %v", err)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := resp.Header.Get("Retry-After")
			retrySeconds, err := strconv.Atoi(retryAfter)
			if err != nil {
				log.Printf("Error while processing retrry-after header: %v", err)
				time.Sleep(time.Minute)
			}
			time.Sleep(time.Duration(retrySeconds) * time.Second)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("Accrual system returned non-200 code: %d %s", resp.StatusCode, (body))
			continue
		}

		var data orderResponse
		err = json.Unmarshal(body, &data)
		if err != nil {
			log.Printf("Error while unmarshalling accrual system's response: %v", err)
			continue
		}

		err = store.ProcessAccrual(data.Order, data.Status, data.Accrual)
		if err != nil {
			log.Printf("Error while processing accrual id db: %v", err)
		}
	}

	return nil
}
