package core

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/devsagul/gophemart/internal/utils"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidHashFormat        = errors.New("ivalid encioded hash format")
	ErrIncompatibleArgonVersion = errors.New("incompatible argon version")
)

type User struct {
	Id           uuid.UUID
	Login        string
	passwordHash string
	Balance      decimal.Decimal
}

func NewUser(login, password string) (*User, error) {
	user := new(User)
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	user.Id = id
	user.Balance = decimal.Zero
	passwordHash, err := generatePasswordHash(password)
	if err != nil {
		return nil, err
	}
	user.Login = login
	user.passwordHash = passwordHash
	return user, nil
}

func (user *User) ValidatePassword(password string) (bool, error) {
	decodedHash, err := decodeHash(user.passwordHash)
	if err != nil {
		return false, err
	}

	hash := argon2.IDKey([]byte(password), decodedHash.salt, decodedHash.iterations, decodedHash.memory, decodedHash.parallelism, decodedHash.keyLength)

	if subtle.ConstantTimeCompare(decodedHash.hash, hash) == 1 {
		return true, nil
	}

	return false, nil
}

type passwordGenerationParams struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLenghth uint32
	keyLength   uint32
}

type passwordHash struct {
	passwordGenerationParams
	salt []byte
	hash []byte
}

func generatePasswordHash(pasword string) (string, error) {
	p := &passwordGenerationParams{
		64 * 1024,
		3,
		2,
		16,
		32,
	}

	salt, err := utils.GenerateRandomBytes(p.saltLenghth)
	if err != nil {
		return "", nil
	}

	hash := argon2.IDKey([]byte(pasword), salt, p.iterations, p.memory, p.parallelism, p.keyLength)

	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, p.memory, p.iterations, p.parallelism, encodedSalt, encodedHash)

	return encoded, nil
}

func decodeHash(encoded string) (*passwordHash, error) {
	vals := strings.Split(encoded, "$")
	if len(vals) != 6 {
		return nil, ErrInvalidHashFormat
	}

	var version int
	_, err := fmt.Sscanf(vals[2], "v=%d", &version)
	if err != nil {
		return nil, err
	}

	if version != argon2.Version {
		return nil, ErrIncompatibleArgonVersion
	}

	params := passwordGenerationParams{}

	_, err = fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &params.memory, &params.iterations, &params.parallelism)
	if err != nil {
		return nil, err
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(vals[4])
	if err != nil {
		return nil, err
	}
	params.saltLenghth = uint32(len(salt))

	hash, err := base64.RawStdEncoding.Strict().DecodeString(vals[5])
	if err != nil {
		return nil, err
	}
	params.keyLength = uint32(len(hash))

	return &passwordHash{params, salt, hash}, nil
}
