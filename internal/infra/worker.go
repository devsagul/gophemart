package infra

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"

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

	if err != nil {
		return err
	}

	for order := range orders {
		// todo manage frequency
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
		if resp.StatusCode != 200 {
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
