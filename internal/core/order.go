//go:generate stringer -type=OrderStatus

package core

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type OrderStatus int

const (
	NEW = OrderStatus(iota + 1)
	PROCESSING
	INVALID
	PROCESSED
)

var ERR_INVALID_ORDER = errors.New("invalid order id")
var ErrUserNotSupplied = errors.New("no user supplied")

type Order struct {
	// todo add json tags
	Id         string
	Status     OrderStatus
	UploadedAt time.Time
	UserId     uuid.UUID
}

func NewOrder(id string, user *User, uploadedAt time.Time) (*Order, error) {
	if len(id) == 0 {
		return nil, ERR_INVALID_ORDER
	}

	if user == nil {
		return nil, ErrUserNotSupplied
	}

	sum := 0
	odd := len(id) % 2

	for i, character := range id {
		if character < '0' || character > '9' {
			return nil, ERR_INVALID_ORDER
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
		return nil, ERR_INVALID_ORDER
	}

	return &Order{
		id,
		NEW,
		uploadedAt,
		user.Id,
	}, nil
}
