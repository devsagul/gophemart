package infra

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
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

func Worker(
	orders <-chan *core.Order,
	apiAddress string,
	store storage.Storage,
) error {
	originalAPIURL, err := url.Parse(apiAddress)

	log.Printf("Original api url: %s", originalAPIURL.String())
	if err != nil {
		return err
	}

	for order := range orders {
		log.Printf("incoming order: %s", order.ID)
		time.Sleep(15)
		// todo manage frequency
		apiURL := *originalAPIURL

		apiURL.Path = path.Join(originalAPIURL.Path, fmt.Sprintf("/api/orders/%s", order.ID))
		log.Printf("Getting order info from %s", apiURL.String())
		resp, err := http.Get(apiURL.String())
		if err != nil {
			log.Printf("Error during orders processing: %v", err)
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			log.Printf("Accrual system returned non-200 code: %d %s", resp.StatusCode, (body))
			continue
		}
		log.Printf("Response form accrual system: %d %s", resp.StatusCode, (body))

		if err != nil {
			log.Printf("Error while reading accrual system's response: %v", err)
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
