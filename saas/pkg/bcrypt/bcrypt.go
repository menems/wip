package bcrypt

import (
	gocrypt "golang.org/x/crypto/bcrypt"
)

// DefaultCost is the bcrypt work factor used when none is specified.
const DefaultCost = gocrypt.DefaultCost

// Hasher implements password hashing and comparison using bcrypt.
// It satisfies both user.PasswordHasher and auth.PasswordComparer.
type Hasher struct {
	cost int
}

// New creates a Hasher with the given bcrypt cost.
func New(cost int) *Hasher {
	return &Hasher{cost: cost}
}

// Hash generates a bcrypt hash of the plaintext password.
func (h *Hasher) Hash(password string) (string, error) {
	b, err := gocrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Compare returns nil if password matches hash, or an error otherwise.
func (h *Hasher) Compare(hash, password string) error {
	return gocrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
