package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/devsagul/gophemart/internal/core"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
)

// todo add context

type postgresStorage struct {
	db  *sql.DB
	ctx context.Context
}

func (store *postgresStorage) CreateKey(key *core.HmacKey) error {
	putQuery, err := store.db.Prepare("INSERT INTO hmac_key(id, sign, expires_at) VALUES($1, $2, $3)")
	if err != nil {
		return err
	}
	_, err = putQuery.Exec(key.Id, key.Sign, key.ExpiresAt)
	return err
}

func (store *postgresStorage) ExtractKey(id uuid.UUID) (*core.HmacKey, error) {
	now := time.Now()

	query, err := store.db.Prepare("SELECT id, sign, expires_at from hmac_key WHERE id = $1 AND expires_at > $2")

	if err != nil {
		return nil, err
	}

	rows, err := query.Query(id, now)
	if err != nil {
		return nil, err
	}
	for rows.Next() {

		var key core.HmacKey
		err = rows.Scan(&key.Id, &key.Sign, &key.ExpiresAt)
		if err != nil {
			return nil, err
		}

		if key.Fresh() {
			return &key, nil
		}
	}

	return nil, &ErrKeyNotFound{id}
}
func (store *postgresStorage) ExtractRandomKey() (*core.HmacKey, error) {
	now := time.Now()

	query, err := store.db.Prepare("SELECT id, sign, expires_at from hmac_key WHERE expires_at > $1 ORDER BY RANDOM()")
	if err != nil {
		return nil, err
	}

	rows, err := query.Query(now)
	if err != nil {
		return nil, err
	}
	for rows.Next() {

		var key core.HmacKey
		err = rows.Scan(&key.Id, &key.Sign, &key.ExpiresAt)
		if err != nil {
			return nil, err
		}

		if key.Fresh() {
			return &key, nil
		}
	}

	return nil, &ErrNoKeys{}
}

func (store *postgresStorage) ExtractAllKeys() (map[uuid.UUID]core.HmacKey, error) {
	keys := make(map[uuid.UUID]core.HmacKey)
	now := time.Now()

	query, err := store.db.Prepare("SELECT id, sign, expires_at from hmac_key WHERE expires_at > $1")

	if err != nil {
		return nil, err
	}

	rows, err := query.Query(now)
	if err != nil {
		return nil, err
	}
	for rows.Next() {

		var key core.HmacKey
		err = rows.Scan(&key.Id, &key.Sign, &key.ExpiresAt)
		if err != nil {
			return nil, err
		}
		keys[key.Id] = key
	}

	return keys, nil
}

// orders
func (store *postgresStorage) CreateOrder(order *core.Order) error {
	tx, err := store.db.Begin()
	defer func() {
		err := tx.Rollback()
		if err != nil {
			if err.Error() != "sql: transaction has already been committed or rolled back" {
				log.Printf("error during transaction rollback: %v", err)
			}
		}
	}()
	if err != nil {
		return err
	}

	query, err := tx.Prepare("SELECT user_id from app_order WHERE id = $1")
	if err != nil {
		return err
	}

	row := query.QueryRow(order.Id)

	var userId uuid.UUID

	err = row.Scan(&userId)

	switch err {
	case nil:
		if userId == order.UserId {
			return &ErrOrderExists{order.Id}
		}

		return &ErrOrderIdCollission{order.Id}
	case sql.ErrNoRows:
	default:
		return err
	}

	putQuery, err := tx.Prepare("INSERT INTO app_order(id, status, uploaded_at, user_id) VALUES($1, $2, $3, $4)")
	if err != nil {
		return err
	}
	_, err = putQuery.Exec(order.Id, order.Status, order.UploadedAt, order.UserId)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (store *postgresStorage) ExtractOrdersByUser(user *core.User) ([]*core.Order, error) {
	userId := user.Id
	orders := []*core.Order{}

	query, err := store.db.Prepare("SELECT id, status, user_id, uploaded_at from app_order WHERE user_id = $1")

	if err != nil {
		return nil, err
	}

	rows, err := query.Query(userId)
	if err != nil {
		return nil, err
	}
	for rows.Next() {

		var order core.Order
		err = rows.Scan(&order.Id, &order.Status, &order.UserId, &order.UploadedAt)
		if err != nil {
			return nil, err
		}
		order.UploadedAt = order.UploadedAt.Local()
		orders = append(orders, &order)
	}

	return orders, nil
}

// users
func (store *postgresStorage) CreateUser(user *core.User) error {
	// we have to check on application error so as not to parse psql error
	tx, err := store.db.Begin()
	defer func() {
		err := tx.Rollback()
		if err != nil {
			if err.Error() != "sql: transaction has already been committed or rolled back" {
				log.Printf("error during transaction rollback: %v", err)
			}
		}
	}()
	if err != nil {
		return err
	}

	query, err := tx.Prepare("SELECT 1 from app_user WHERE login = $1")
	if err != nil {
		return err
	}

	rows, err := query.Query(user.Login)
	if err != nil {
		return err
	}

	for rows.Next() {
		return &ErrConflictingUserLogin{user.Login}
	}

	putQuery, err := tx.Prepare("INSERT INTO app_user(id, login, password_hash, balance) VALUES($1, $2, $3, $4)")
	if err != nil {
		return err
	}
	_, err = putQuery.Exec(user.Id, user.Login, user.PasswordHash, user.Balance)
	if err != nil {
		return err
	}
	err = tx.Commit()
	return err
}

func (store *postgresStorage) ExtractUser(login string) (*core.User, error) {
	query, err := store.db.Prepare("SELECT id, login, password_hash, balance from app_user WHERE login = $1")
	if err != nil {
		return nil, err
	}

	rows, err := query.Query(login)
	if err != nil {
		return nil, err
	}
	for rows.Next() {

		var user core.User
		err = rows.Scan(&user.Id, &user.Login, &user.PasswordHash, &user.Balance)
		if err != nil {
			return nil, err
		}

		return &user, nil
	}

	return nil, &ErrUserNotFound{login}
}

func (store *postgresStorage) ExtractUserById(id uuid.UUID) (*core.User, error) {
	query, err := store.db.Prepare("SELECT id, login, password_hash, balance from app_user WHERE id = $1")
	if err != nil {
		return nil, err
	}

	rows, err := query.Query(id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {

		var user core.User
		err = rows.Scan(&user.Id, &user.Login, &user.PasswordHash, &user.Balance)
		if err != nil {
			return nil, err
		}

		return &user, nil
	}

	return nil, &ErrUserNotFoundById{id}
}

// withdrawals
func (store *postgresStorage) CreateWithdrawal(withdrawal *core.Withdrawal, order *core.Order) error {
	tx, err := store.db.Begin()
	defer func() {
		err := tx.Rollback()
		if err != nil {
			if err.Error() != "sql: transaction has already been committed or rolled back" {
				log.Printf("error during transaction rollback: %v", err)
			}
		}
	}()
	if err != nil {
		return err
	}

	query, err := tx.Prepare("SELECT user_id from app_order WHERE id = $1")
	if err != nil {
		return err
	}

	row := query.QueryRow(order.Id)

	var userId uuid.UUID

	err = row.Scan(&userId)

	switch err {
	case nil:
		if userId == order.UserId {
			return &ErrOrderExists{order.Id}
		}

		return &ErrOrderIdCollission{order.Id}
	case sql.ErrNoRows:
	default:
		return err
	}

	putQuery, err := tx.Prepare("INSERT INTO app_order(id, status, uploaded_at, user_id) VALUES($1, $2, $3, $4)")
	if err != nil {
		return err
	}
	_, err = putQuery.Exec(order.Id, order.Status, order.UploadedAt, order.UserId)
	if err != nil {
		return err
	}

	selectQuery, err := tx.Prepare("SELECT app_user.balance FROM app_user WHERE id = $1 FOR UPDATE")
	if err != nil {
		log.Printf("select %v", err)
		return err
	}

	var balance decimal.Decimal
	row = selectQuery.QueryRow(order.UserId)
	err = row.Scan(&balance)
	if err != nil {
		return err
	}
	if balance.LessThan(withdrawal.Sum) {
		return &ErrBalanceExceeded{}
	}
	putQuery, err = tx.Prepare("INSERT INTO withdrawal(id, order_id, processed_at, withdrawal_sum) VALUES($1, $2, $3, $4)")
	if err != nil {
		log.Printf("put %v", err)
		return err
	}
	_, err = putQuery.Exec(withdrawal.Id, withdrawal.OrderId, withdrawal.ProcessedAt, withdrawal.Sum)
	if err != nil {
		return err
	}

	updateQuery, err := tx.Prepare("UPDATE app_user SET balance = $1 WHERE id = $2")
	if err != nil {
		log.Printf("update %v", err)
		return err
	}
	_, err = updateQuery.Exec(balance.Sub(withdrawal.Sum), order.UserId)
	if err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

func (store *postgresStorage) ExtractWithdrawalsByUser(user *core.User) ([]*core.Withdrawal, error) {
	var withdrawals []*core.Withdrawal

	selectQuery, err := store.db.Prepare("SELECT withdrawal.id, order_id, withdrawal_sum, processed_at FROM withdrawal INNER JOIN app_order on withdrawal.order_id = app_order.id INNER JOIN app_user ON app_order.user_id = app_user.id WHERE app_user.id = $1 ORDER BY withdrawal.processed_at")
	if err != nil {
		log.Printf("select %v", err)
		return withdrawals, err
	}

	rows, err := selectQuery.Query(user.Id)

	if err != nil {
		return withdrawals, err
	}

	for rows.Next() {
		var withdrawal core.Withdrawal
		err := rows.Scan(&withdrawal.Id, &withdrawal.OrderId, &withdrawal.Sum, &withdrawal.ProcessedAt)

		if err != nil {
			return []*core.Withdrawal{}, err
		}

		withdrawal.ProcessedAt = withdrawal.ProcessedAt.Local()

		withdrawals = append(withdrawals, &withdrawal)
	}

	return withdrawals, nil
}

func (store *postgresStorage) TotalWithdrawnSum(user *core.User) (decimal.Decimal, error) {
	query, err := store.db.Prepare("SELECT COALESCE(SUM(withdrawal_sum), 0) FROM withdrawal INNER JOIN app_order ON withdrawal.order_id = app_order.id WHERE app_order.user_id = $1")
	if err != nil {
		return decimal.Zero, err
	}

	rows, err := query.Query(user.Id)
	if err != nil {
		return decimal.Zero, err
	}
	for rows.Next() {
		var sum decimal.Decimal
		err = rows.Scan(&sum)
		if err != nil {
			return decimal.Zero, err
		}

		return sum, nil
	}

	return decimal.Zero, errors.New("no rows selected")
}

func (store *postgresStorage) ProcessAccrual(orderId string, status string, sum *decimal.Decimal) error {
	if status == "REGISTERED" {
		status = core.NEW
	}

	if status != core.NEW || status != core.PROCESSED || status != core.INVALID || status != core.PROCESSED {
		return fmt.Errorf("invalid order status: %v", status)
	}

	tx, err := store.db.Begin()
	defer func() {
		err := tx.Rollback()
		if err != nil {
			if err.Error() != "sql: transaction has already been committed or rolled back" {
				log.Printf("error during transaction rollback: %v", err)
			}
		}
	}()
	if err != nil {
		return err
	}

	query, err := tx.Prepare("UPDATE app_order SET status = $2 WHERE id = $1")
	if err != nil {
		return err
	}
	res, err := query.Exec(orderId, status)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("expected one row to be affected, got %d", n)
	}

	if sum != nil {
		query, err = tx.Prepare("UPDATE app_order SET accrual = $2 WHERE id = $1")
		if err != nil {
			return err
		}
		res, err = query.Exec(orderId, *sum)
		if err != nil {
			return err
		}
		n, err = res.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return fmt.Errorf("expected one row to be affected, got %d", n)
		}

		query, err = tx.Prepare("UPDATE app_user SET balance = balance + $2 FROM app_order WHERE app_order.id = $1 AND app_user.id = app_order.user_id")
		if err != nil {
			return err
		}
		res, err = query.Exec(orderId, *sum)
		if err != nil {
			return err
		}
		n, err = res.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return fmt.Errorf("expected one row to be affected, got %d", n)
		}
	}

	return tx.Commit()
}

func (store *postgresStorage) Ping(ctx context.Context) error {
	return store.db.PingContext(ctx)
}

func (store *postgresStorage) WithContext(ctx context.Context) Storage {
	newStore := *store
	newStore.ctx = ctx
	return &newStore
}

func NewPostgresStorage(dsn string) (Storage, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS hmac_key (id UUID PRIMARY KEY, sign BYTEA NOT NULL, expires_at TIMESTAMP WITH TIME ZONE NOT NULL)")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS expires_index ON hmac_key (expires_at)")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS app_user (id UUID PRIMARY KEY, login TEXT NOT NULL, password_hash TEXT NOT NULL, balance NUMERIC NOT NULL DEFAULT 0)")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS app_order (id TEXT PRIMARY KEY, status VARCHAR(255) NOT NULL, uploaded_at TIMESTAMP WITH TIME ZONE NOT NULL, user_id UUID NOT NULL, accrual NUMERIC NOT NULL DEFAULT 0, CONSTRAINT fk_user FOREIGN KEY(user_id) REFERENCES app_user(id))")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS uploaded_at_index ON app_order (uploaded_at)")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS user_index ON app_order (user_id)")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS status_index ON app_order (status)")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS withdrawal (id UUID PRIMARY KEY, order_id TEXT NOT NULL, processed_at TIMESTAMP WITH TIME ZONE NOT NULL, withdrawal_sum NUMERIC NOT NULL DEFAULT 0, CONSTRAINT fk_order FOREIGN KEY(order_id) REFERENCES app_order(id))")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS order_index ON withdrawal (order_id)")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS processed_index ON withdrawal (processed_at)")
	if err != nil {
		return nil, err
	}

	p := new(postgresStorage)
	p.db = db
	return p, nil
}