package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
)

// Access tokens are deliberately short. Sixty seconds is long enough that a
// single test never notices and short enough that a suite of any size will,
// which is exactly the failure this is here to reproduce.
const (
	AccessLifetime  = 60 * time.Second
	RefreshLifetime = 30 * time.Minute
)

// Claims is what the playground puts in a token.
type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	Kind     string `json:"kind"` // access or refresh
	jwt.RegisteredClaims
}

// Issue mints a signed token. The clock is the session's, so freezing or
// advancing time changes what a token means without anybody waiting.
func (s *Store) Issue(now time.Time, login Login, kind string) (string, error) {
	lifetime := AccessLifetime
	if kind == "refresh" {
		lifetime = RefreshLifetime
	}

	id := fmt.Sprintf("%s-%s-%d", login.Username, kind, now.UnixNano())
	claims := Claims{
		Username: login.Username,
		Role:     login.Role,
		Kind:     kind,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        id,
			Subject:   login.Username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(lifetime)),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

// ErrExpired is returned separately from every other rejection, because "your
// token ran out" and "your token is wrong" call for completely different
// responses and a test needs to tell them apart.
var ErrExpired = errors.New("token expired")

// Verify checks a token against the session's key and clock.
func (s *Store) Verify(now time.Time, token string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return s.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		// The session's clock rather than the wall clock, so advancing time
		// through the control plane expires tokens immediately.
		jwt.WithTimeFunc(func() time.Time { return now }),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpired
		}
		return nil, err
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok {
		return nil, errors.New("unexpected claims")
	}
	if s.Revoked(claims.ID) {
		return nil, errors.New("token revoked")
	}
	return claims, nil
}

// TOTPCode computes the code valid at a given instant, so the control plane
// can hand a test the answer rather than leaving it locked out.
func (s *Store) TOTPCode(now time.Time) (string, error) {
	return totp.GenerateCode(s.totpSecret, now)
}

// CheckTOTP validates a submitted code. One period of skew either way is
// allowed, which is what real implementations do and what stops a test failing
// because it crossed a thirty-second boundary mid-request.
func (s *Store) CheckTOTP(now time.Time, code string) bool {
	for _, offset := range []time.Duration{0, -30 * time.Second, 30 * time.Second} {
		expected, err := s.TOTPCode(now.Add(offset))
		if err == nil && expected == code {
			return true
		}
	}
	return false
}

// TOTPURI is the provisioning URI an authenticator app would scan.
func (s *Store) TOTPURI(username string) string {
	return fmt.Sprintf(
		"otpauth://totp/testground:%s?secret=%s&issuer=testground&algorithm=SHA1&digits=6&period=30",
		username, s.totpSecret,
	)
}
