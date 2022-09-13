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

type postgresStorage struct {
	db  *sql.DB
	ctx context.Context
}

// Persist new hmac key within the storage
func (store *postgresStorage) CreateKey(key *core.HmacKey) error {
	putQuery, err := store.db.Prepare("INSERT INTO hmac_key(id, sign, expires_at) VALUES($1, $2, $3)")
	if err != nil {
		return err
	}
	_, err = putQuery.ExecContext(store.ctx, key.ID, key.Sign, key.ExpiresAt)
	return err
}

// extract a key by id
func (store *postgresStorage) ExtractKey(id uuid.UUID) (*core.HmacKey, error) {
	now := time.Now()

	query, err := store.db.PrepareContext(store.ctx, "SELECT id, sign, expires_at from hmac_key WHERE id = $1 AND expires_at > $2")

	if err != nil {
		return nil, err
	}

	rows, err := query.QueryContext(store.ctx, id, now)
	if err != nil {
		return nil, err
	}
	for rows.Next() {

		var key core.HmacKey
		err = rows.Scan(&key.ID, &key.Sign, &key.ExpiresAt)
		if err != nil {
			return nil, err
		}

		if key.Fresh() {
			return &key, nil
		}
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return nil, &ErrKeyNotFound{id}
}

// extract random valid fresh key (if any)
func (store *postgresStorage) ExtractRandomKey() (*core.HmacKey, error) {
	now := time.Now()

	query, err := store.db.PrepareContext(store.ctx, "SELECT id, sign, expires_at from hmac_key WHERE expires_at > $1 ORDER BY RANDOM()")
	if err != nil {
		return nil, err
	}

	rows, err := query.QueryContext(store.ctx, now)
	if err != nil {
		return nil, err
	}
	for rows.Next() {

		var key core.HmacKey
		err = rows.Scan(&key.ID, &key.Sign, &key.ExpiresAt)
		if err != nil {
			return nil, err
		}

		if key.Fresh() {
			return &key, nil
		}
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return nil, &ErrNoKeys{}
}

// extract all keys
func (store *postgresStorage) ExtractAllKeys() (map[uuid.UUID]core.HmacKey, error) {
	keys := make(map[uuid.UUID]core.HmacKey)
	now := time.Now()

	query, err := store.db.PrepareContext(store.ctx, "SELECT id, sign, expires_at from hmac_key WHERE expires_at > $1")

	if err != nil {
		return nil, err
	}

	rows, err := query.QueryContext(store.ctx, now)
	if err != nil {
		return nil, err
	}
	for rows.Next() {

		var key core.HmacKey
		err = rows.Scan(&key.ID, &key.Sign, &key.ExpiresAt)
		if err != nil {
			return nil, err
		}
		keys[key.ID] = key
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return keys, nil
}

// orders

// Persist new order item
func (store *postgresStorage) CreateOrder(order *core.Order) error {
	tx, err := store.db.BeginTx(store.ctx, nil)
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

	query, err := tx.PrepareContext(store.ctx, "SELECT user_id from app_order WHERE id = $1")
	if err != nil {
		return err
	}

	row := query.QueryRowContext(store.ctx, order.ID)

	var userID uuid.UUID

	err = row.Scan(&userID)

	switch err {
	case nil:
		if userID == order.UserID {
			return &ErrOrderExists{order.ID}
		}

		return &ErrOrderIDCollission{order.ID}
	case sql.ErrNoRows:
	default:
		return err
	}

	putQuery, err := tx.PrepareContext(store.ctx, "INSERT INTO app_order(id, status, uploaded_at, user_id) VALUES($1, $2, $3, $4)")
	if err != nil {
		return err
	}
	_, err = putQuery.Exec(order.ID, order.Status, order.UploadedAt, order.UserID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Extract all orders by user
func (store *postgresStorage) ExtractOrdersByUser(user *core.User) ([]*core.Order, error) {
	userID := user.ID
	orders := []*core.Order{}

	query, err := store.db.PrepareContext(store.ctx, "SELECT id, status, user_id, uploaded_at, accrual from app_order WHERE user_id = $1")

	if err != nil {
		return nil, err
	}

	rows, err := query.QueryContext(store.ctx, userID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {

		var order core.Order

		var accrual decimal.NullDecimal

		err = rows.Scan(&order.ID, &order.Status, &order.UserID, &order.UploadedAt, &accrual)
		if err != nil {
			return nil, err
		}

		if accrual.Valid {
			order.Accrual = &accrual.Decimal
		} else {
			order.Accrual = nil
		}

		order.UploadedAt = order.UploadedAt.Local()
		orders = append(orders, &order)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return orders, nil
}

// Extract all unterminated orders for all users
func (store *postgresStorage) ExtractUnterminatedOrders() ([]*core.Order, error) {
	orders := []*core.Order{}
	query, err := store.db.PrepareContext(store.ctx, "SELECT id, status, user_id, uploaded_at from app_order WHERE status != $1 AND status != $2 ORDER BY uploaded_at")

	if err != nil {
		return nil, err
	}

	rows, err := query.QueryContext(store.ctx, core.PROCESSED, core.INVALID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {

		var order core.Order
		err = rows.Scan(&order.ID, &order.Status, &order.UserID, &order.UploadedAt)
		if err != nil {
			return nil, err
		}
		order.UploadedAt = order.UploadedAt.Local()
		orders = append(orders, &order)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return orders, nil
}

// users

// Persist new user
func (store *postgresStorage) CreateUser(user *core.User) error {
	// we have to check on application error so as not to parse psql error
	tx, err := store.db.BeginTx(store.ctx, nil)
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

	query, err := tx.PrepareContext(store.ctx, "SELECT 1 from app_user WHERE login = $1")
	if err != nil {
		return err
	}

	rows, err := query.QueryContext(store.ctx, user.Login)
	if err != nil {
		return err
	}

	for rows.Next() {
		return &ErrConflictingUserLogin{user.Login}
	}
	err = rows.Err()
	if err != nil {
		return err
	}

	putQuery, err := tx.PrepareContext(store.ctx, "INSERT INTO app_user(id, login, password_hash, balance) VALUES($1, $2, $3, $4)")
	if err != nil {
		return err
	}
	_, err = putQuery.Exec(user.ID, user.Login, user.PasswordHash, user.Balance)
	if err != nil {
		return err
	}
	err = tx.Commit()
	return err
}

// Extract user by login
func (store *postgresStorage) ExtractUser(login string) (*core.User, error) {
	query, err := store.db.PrepareContext(store.ctx, "SELECT id, login, password_hash, balance from app_user WHERE login = $1")
	if err != nil {
		return nil, err
	}

	row := query.QueryRowContext(store.ctx, login)
	switch row.Err() {
	case sql.ErrNoRows:
		return nil, &ErrUserNotFound{login}
	case nil:
	default:
		return nil, err
	}

	var user core.User
	err = row.Scan(&user.ID, &user.Login, &user.PasswordHash, &user.Balance)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// Extract user by id
func (store *postgresStorage) ExtractUserByID(id uuid.UUID) (*core.User, error) {
	query, err := store.db.PrepareContext(store.ctx, "SELECT id, login, password_hash, balance from app_user WHERE id = $1")
	if err != nil {
		return nil, err
	}

	row := query.QueryRowContext(store.ctx, id)
	switch row.Err() {
	case sql.ErrNoRows:
		return nil, &ErrUserNotFoundByID{id}
	case nil:
	default:
		return nil, err
	}

	var user core.User
	err = row.Scan(&user.ID, &user.Login, &user.PasswordHash, &user.Balance)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// withdrawals

// Persist new withdrawal item
func (store *postgresStorage) CreateWithdrawal(withdrawal *core.Withdrawal, order *core.Order) error {
	tx, err := store.db.BeginTx(store.ctx, nil)
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

	query, err := tx.PrepareContext(store.ctx, "SELECT user_id from app_order WHERE id = $1")
	if err != nil {
		return err
	}

	row := query.QueryRowContext(store.ctx, order.ID)

	var userID uuid.UUID

	err = row.Scan(&userID)

	switch err {
	case nil:
		if userID == order.UserID {
			return &ErrOrderExists{order.ID}
		}

		return &ErrOrderIDCollission{order.ID}
	case sql.ErrNoRows:
	default:
		return err
	}

	putQuery, err := tx.PrepareContext(store.ctx, "INSERT INTO app_order(id, status, uploaded_at, user_id) VALUES($1, $2, $3, $4)")
	if err != nil {
		return err
	}
	_, err = putQuery.ExecContext(store.ctx, order.ID, order.Status, order.UploadedAt, order.UserID)
	if err != nil {
		return err
	}

	selectQuery, err := tx.PrepareContext(store.ctx, "SELECT app_user.balance FROM app_user WHERE id = $1 FOR UPDATE")
	if err != nil {
		return err
	}

	var balance decimal.Decimal
	row = selectQuery.QueryRowContext(store.ctx, order.UserID)
	err = row.Scan(&balance)
	if err != nil {
		return err
	}
	if balance.LessThan(withdrawal.Sum) {
		return &ErrBalanceExceeded{}
	}
	putQuery, err = tx.PrepareContext(store.ctx, "INSERT INTO withdrawal(id, order_id, processed_at, withdrawal_sum) VALUES($1, $2, $3, $4)")
	if err != nil {
		log.Printf("put %v", err)
		return err
	}
	_, err = putQuery.ExecContext(store.ctx, withdrawal.ID, withdrawal.OrderID, withdrawal.ProcessedAt, withdrawal.Sum)
	if err != nil {
		return err
	}

	updateQuery, err := tx.PrepareContext(store.ctx, "UPDATE app_user SET balance = $1 WHERE id = $2")
	if err != nil {
		return err
	}
	_, err = updateQuery.ExecContext(store.ctx, balance.Sub(withdrawal.Sum), order.UserID)
	if err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

// Extract withdrawals by user
func (store *postgresStorage) ExtractWithdrawalsByUser(user *core.User) ([]*core.Withdrawal, error) {
	var withdrawals []*core.Withdrawal

	selectQuery, err := store.db.PrepareContext(store.ctx, "SELECT withdrawal.id, order_id, withdrawal_sum, processed_at FROM withdrawal INNER JOIN app_order on withdrawal.order_id = app_order.id INNER JOIN app_user ON app_order.user_id = app_user.id WHERE app_user.id = $1 ORDER BY withdrawal.processed_at")
	if err != nil {
		log.Printf("select %v", err)
		return withdrawals, err
	}

	rows, err := selectQuery.QueryContext(store.ctx, user.ID)

	if err != nil {
		return withdrawals, err
	}

	for rows.Next() {
		var withdrawal core.Withdrawal
		err := rows.Scan(&withdrawal.ID, &withdrawal.OrderID, &withdrawal.Sum, &withdrawal.ProcessedAt)

		if err != nil {
			return []*core.Withdrawal{}, err
		}

		withdrawal.ProcessedAt = withdrawal.ProcessedAt.Local()

		withdrawals = append(withdrawals, &withdrawal)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return withdrawals, nil
}

// Calculate total withdrawn sum for user
func (store *postgresStorage) TotalWithdrawnSum(user *core.User) (decimal.Decimal, error) {
	query, err := store.db.PrepareContext(store.ctx, "SELECT COALESCE(SUM(withdrawal_sum), 0) FROM withdrawal INNER JOIN app_order ON withdrawal.order_id = app_order.id WHERE app_order.user_id = $1")
	if err != nil {
		return decimal.Zero, err
	}

	row := query.QueryRowContext(store.ctx, user.ID)
	switch row.Err() {
	case sql.ErrNoRows:
		return decimal.Zero, errors.New("no rows selected")
	case nil:
	default:
		return decimal.Zero, err
	}

	var sum decimal.Decimal
	err = row.Scan(&sum)
	if err != nil {
		return decimal.Zero, err
	}

	return sum, nil
}

// Register new accrual within the storage
func (store *postgresStorage) ProcessAccrual(orderID string, status string, sum *decimal.Decimal) error {
	if status == "REGISTERED" {
		status = core.NEW
	}

	if status != core.NEW && status != core.PROCESSING && status != core.INVALID && status != core.PROCESSED {
		return fmt.Errorf("invalid order status: %s", status)
	}

	tx, err := store.db.BeginTx(store.ctx, nil)
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

	query, err := tx.PrepareContext(store.ctx, "UPDATE app_order SET status = $2 WHERE id = $1 AND status != $3 AND status != $4")
	if err != nil {
		return err
	}
	res, err := query.ExecContext(store.ctx, orderID, status, core.PROCESSED, core.INVALID)
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
		query, err = tx.PrepareContext(store.ctx, "UPDATE app_order SET accrual = $2 WHERE id = $1")
		if err != nil {
			return err
		}
		res, err = query.ExecContext(store.ctx, orderID, *sum)
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

		query, err = tx.PrepareContext(store.ctx, "UPDATE app_user SET balance = balance + $2 FROM app_order WHERE app_order.id = $1 AND app_user.id = app_order.user_id")
		if err != nil {
			return err
		}
		res, err = query.ExecContext(store.ctx, orderID, *sum)
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

// Checks if storage is awailable
func (store *postgresStorage) Ping(ctx context.Context) error {
	return store.db.PingContext(ctx)
}

// Constructs new storage object with given context
func (store *postgresStorage) WithContext(ctx context.Context) Storage {
	newStore := *store
	newStore.ctx = ctx
	return &newStore
}

// Create new postgres storage with given DSN
func NewPostgresStorage(dsn string) (*postgresStorage, error) {
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

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS app_order (id TEXT PRIMARY KEY, status VARCHAR(255) NOT NULL, uploaded_at TIMESTAMP WITH TIME ZONE NOT NULL, user_id UUID NOT NULL, accrual NUMERIC NULL DEFAULT NULL, CONSTRAINT fk_user FOREIGN KEY(user_id) REFERENCES app_user(id))")
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
	p.ctx = context.Background()
	return p, nil
}
