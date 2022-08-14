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
	Status  string           `json:"order"`
	Accrual *decimal.Decimal `json:"accrual"`
}

func Worker(
	orders <-chan *core.Order,
	apiAddress string,
	store storage.Storage,
) error {
	originalApiUrl, err := url.Parse(apiAddress)
	if err != nil {
		return err
	}

	for order := range orders {
		// todo manage frequency
		apiUrl := originalApiUrl
		apiUrl.Path = path.Join(originalApiUrl.Path, fmt.Sprintf("/api/orders/%s", order.Id))
		log.Printf("Getting order info from %s", apiUrl.String())
		resp, err := http.Get(apiUrl.String())
		if err != nil {
			log.Printf("Error during orders processing: %v", err)
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error while reading accrual system's response: %v", err)
			continue
		}

		var data orderResponse
		err = json.Unmarshal(body, &data)
		if err != nil {
			log.Printf("Error while unmarshalling accrual system's response: %v", err)
			log.Printf("Response: %s", string(body))
			continue
		}

		err = store.ProcessAccrual(data.Order, data.Status, data.Accrual)
		if err != nil {
			log.Printf("Error while processing accrual id db: %v", err)
		}
	}

	return nil
}
