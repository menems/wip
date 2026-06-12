// Package auth contains the authentication domain: service, ports, and token contracts.
package auth

// TokenService is the outbound port for token generation and validation.
// Implementations live in adapter/jwt.
type TokenService interface {
	Generate(userID, email string) (string, error)
	Validate(token string) (userID, email string, err error)
}
