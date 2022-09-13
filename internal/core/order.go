//go:generate stringer -type=OrderStatus

package core

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type OrderStatus = string

const (
	NEW        = "NEW"
	PROCESSING = "PROCESSING"
	INVALID    = "INVALID"
	PROCESSED  = "PROCESSED"
)

type ErrInvalidOrder struct {
	orderID string
}

func (err *ErrInvalidOrder) Error() string {
	return fmt.Sprintf("invalid order number: %s", err.orderID)
}

type Order struct {
	ID         string           `json:"number"`
	Status     OrderStatus      `json:"status"`
	UploadedAt time.Time        `json:"uploaded_at"`
	UserID     uuid.UUID        `json:"-"`
	Accrual    *decimal.Decimal `json:"accrual,omitempty"`
}

func NewOrder(id string, user *User, uploadedAt time.Time) (*Order, error) {
	if len(id) == 0 {
		return nil, &ErrInvalidOrder{id}
	}

	sum := 0
	odd := len(id) % 2

	for i, character := range id {
		if character < '0' || character > '9' {
			return nil, &ErrInvalidOrder{id}
		}
		digit := int(character - '0')
		if i%2 == odd {
			digit = digit * 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}

	if sum%10 != 0 {
		return nil, &ErrInvalidOrder{id}
	}

	return &Order{
		id,
		NEW,
		uploadedAt,
		user.ID,
		nil,
	}, nil
}
