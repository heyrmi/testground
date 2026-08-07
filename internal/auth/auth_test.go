package auth

import (
	"testing"
	"time"

	"github.com/heyrmi/testground/internal/session"
)

func newStore(t *testing.T, seed uint64) (*Store, *session.Session) {
	t.Helper()
	sess := session.NewStore(session.Options{Seed: seed}).Create()
	return For(sess), sess
}

func TestSecretsAreDeterministicPerSeed(t *testing.T) {
	first, _ := newStore(t, 42)
	second, _ := newStore(t, 42)
	other, _ := newStore(t, 43)

	if first.SecretHex() != second.SecretHex() {
		t.Error("the same seed produced different signing keys")
	}
	if first.TOTPSecret() != second.TOTPSecret() {
		t.Error("the same seed produced different TOTP secrets")
	}
	if first.SecretHex() == other.SecretHex() {
		t.Error("two seeds produced the same signing key")
	}
}

func TestTwoSessionsHaveIndependentLogins(t *testing.T) {
	alice, _ := newStore(t, 42)
	bob, _ := newStore(t, 42)
	now := time.Now()

	if _, _, ok := alice.Attempt(now, "admin", "admin123"); !ok {
		t.Fatal("alice could not log in")
	}
	if bob.Current() != nil {
		t.Fatal("bob is logged in because alice was")
	}
}

func TestThrottleAfterTooManyFailures(t *testing.T) {
	store, _ := newStore(t, 42)
	now := time.Now()

	for i := range MaxAttempts {
		_, throttled, ok := store.Attempt(now, "admin", "wrong")
		if ok {
			t.Fatalf("attempt %d succeeded with a wrong password", i+1)
		}
		if throttled {
			t.Fatalf("throttled at attempt %d, before the limit", i+1)
		}
	}

	// The right password now, and it must still be refused: the throttle is
	// about the attempts, not about the credentials.
	_, throttled, ok := store.Attempt(now, "admin", "admin123")
	if ok || !throttled {
		t.Fatalf("after %d failures: ok=%v throttled=%v, want false true", MaxAttempts, ok, throttled)
	}

	// Measured on the session clock, so a test advances past it rather than
	// waiting thirty seconds.
	later := now.Add(Lockout + time.Second)
	if _, _, ok := store.Attempt(later, "admin", "admin123"); !ok {
		t.Fatal("the throttle did not lift")
	}
}

func TestSuccessClearsTheCounter(t *testing.T) {
	store, _ := newStore(t, 42)
	now := time.Now()

	store.Attempt(now, "admin", "wrong")
	store.Attempt(now, "admin", "wrong")
	if _, _, ok := store.Attempt(now, "admin", "admin123"); !ok {
		t.Fatal("could not log in")
	}

	if count, _ := store.Attempts(); count != 0 {
		t.Fatalf("%d failures still counted after a success", count)
	}
}

func TestTwoFactorLoginIsPendingUntilCompleted(t *testing.T) {
	store, _ := newStore(t, 42)
	now := time.Now()

	login, _, ok := store.Attempt(now, "twofactor", "twofactor123")
	if !ok {
		t.Fatal("the password was refused")
	}
	if !login.Pending {
		t.Fatal("a two-factor account logged straight in on the password alone")
	}

	if !store.CompleteSecondFactor() {
		t.Fatal("could not complete the second factor")
	}
	if store.Current().Pending {
		t.Fatal("still pending after completing")
	}
}

func TestTOTPCodesVerify(t *testing.T) {
	store, _ := newStore(t, 42)
	now := time.Now()

	code, err := store.TOTPCode(now)
	if err != nil {
		t.Fatalf("generating a code: %v", err)
	}
	if !store.CheckTOTP(now, code) {
		t.Fatal("the code this store generated did not verify against it")
	}
	if store.CheckTOTP(now, "000000") {
		t.Error("an obviously wrong code was accepted")
	}

	// A code from two periods ago is outside the allowed skew.
	stale, _ := store.TOTPCode(now.Add(-90 * time.Second))
	if store.CheckTOTP(now, stale) {
		t.Error("a code from ninety seconds ago was accepted")
	}
}

func TestAccessTokensExpireOnTheSessionClock(t *testing.T) {
	store, _ := newStore(t, 42)
	now := time.Now()
	login := Login{Username: "admin", Role: "admin"}

	token, err := store.Issue(now, login, "access")
	if err != nil {
		t.Fatalf("issuing: %v", err)
	}
	if _, err := store.Verify(now, token); err != nil {
		t.Fatalf("a fresh token did not verify: %v", err)
	}

	// No sleeping. Moving the clock is what a test does instead.
	if _, err := store.Verify(now.Add(AccessLifetime+time.Second), token); err != ErrExpired {
		t.Fatalf("after the lifetime: %v, want ErrExpired", err)
	}
}

func TestExpiryIsDistinguishableFromEveryOtherRejection(t *testing.T) {
	store, _ := newStore(t, 42)
	other, _ := newStore(t, 99)
	now := time.Now()

	// Signed by a different session, so the signature is wrong rather than old.
	foreign, _ := other.Issue(now, Login{Username: "admin"}, "access")
	if _, err := store.Verify(now, foreign); err == ErrExpired {
		t.Fatal("a wrongly signed token was reported as expired")
	}
	if _, err := store.Verify(now, "not-a-token"); err == ErrExpired {
		t.Fatal("nonsense was reported as expired")
	}
}

func TestRevocationMakesLogoutMeanSomething(t *testing.T) {
	store, _ := newStore(t, 42)
	now := time.Now()

	token, _ := store.Issue(now, Login{Username: "admin", Role: "admin"}, "access")
	claims, err := store.Verify(now, token)
	if err != nil {
		t.Fatalf("verifying: %v", err)
	}

	store.Revoke(claims.ID)
	if _, err := store.Verify(now, token); err == nil {
		t.Fatal("a revoked token still verified, which makes logging out cosmetic")
	}
}

// Nothing about checking a signed token reaches back to the server, so unless
// the logout withdraws the ids as well as the login, a bearer token taken
// before it goes on working afterwards and the logout is browser-side theatre.
func TestLogOutRevokesTokensIssuedBeforeIt(t *testing.T) {
	store, _ := newStore(t, 42)
	now := time.Now()

	login, _, ok := store.Attempt(now, "admin", "admin123")
	if !ok {
		t.Fatal("could not log in")
	}
	access, _ := store.Issue(now, *login, "access")
	refresh, _ := store.Issue(now, *login, "refresh")
	if _, err := store.Verify(now, access); err != nil {
		t.Fatalf("a fresh access token did not verify: %v", err)
	}

	store.LogOut()

	if _, err := store.Verify(now, access); err == nil {
		t.Error("an access token minted before the logout still verified after it")
	}
	// The refresh token has to go with it, or one request buys the whole login
	// back and the logout means nothing again.
	if _, err := store.Verify(now, refresh); err == nil {
		t.Error("a refresh token outlived the logout")
	}
	// Revoked and expired are answered differently -- a suite meeting the
	// second one refreshes, which would be the wrong move here.
	if _, err := store.Verify(now, access); err == ErrExpired {
		t.Error("a revoked token was reported as expired")
	}
}

// Tokens issued after logging back in are the caller's to use. The id carries
// the instant it was minted at and the session clock can be frozen, so this is
// also the case where the new token is byte for byte the one just revoked.
func TestTokensIssuedAfterALogOutStillWork(t *testing.T) {
	store, _ := newStore(t, 42)
	now := time.Now()
	login := Login{Username: "admin", Role: "admin"}

	first, _ := store.Issue(now, login, "access")
	store.LogOut()
	if _, err := store.Verify(now, first); err == nil {
		t.Fatal("the logout did not revoke the token issued before it")
	}

	second, _ := store.Issue(now, login, "access")
	if _, err := store.Verify(now, second); err != nil {
		t.Fatalf("a token issued after logging back in was dead on arrival: %v", err)
	}
}

// Two workers on one seed share a signing key by design, so nothing in the
// token itself tells them apart. Revocation must not be the thing that leaks:
// one worker logging out cannot start rejecting another's requests.
func TestRevocationIsScopedToItsSession(t *testing.T) {
	store := session.NewStore(session.Options{Seed: 42})
	alice := For(store.Open("worker-one"))
	bob := For(store.Open("worker-two"))
	now := time.Now()

	token, _ := alice.Issue(now, Login{Username: "admin", Role: "admin"}, "access")
	alice.LogOut()

	if _, err := bob.Verify(now, token); err != nil {
		t.Fatalf("one worker's logout rejected a token in another: %v", err)
	}
}

// The refresh challenge's released path, which nothing here may disturb: sixty
// seconds, then expiry rather than rejection, then a refresh that works. No
// logout happens in it, so revocation has to stay entirely out of the way.
func TestRefreshingAcrossExpiryIsUnaffected(t *testing.T) {
	store, _ := newStore(t, 42)
	now := time.Now()

	login, _, ok := store.Attempt(now, "user", "user123")
	if !ok {
		t.Fatal("could not log in")
	}
	access, _ := store.Issue(now, *login, "access")
	refresh, _ := store.Issue(now, *login, "refresh")

	if _, err := store.Verify(now, access); err != nil {
		t.Fatalf("the first access token did not verify: %v", err)
	}

	later := now.Add(AccessLifetime + time.Second)
	if _, err := store.Verify(later, access); err != ErrExpired {
		t.Fatalf("past the lifetime: %v, want ErrExpired", err)
	}

	claims, err := store.Verify(later, refresh)
	if err != nil {
		t.Fatalf("the refresh token did not outlive the access token: %v", err)
	}
	renewed, _ := store.Issue(later, Login{Username: claims.Username, Role: claims.Role}, "access")
	if _, err := store.Verify(later, renewed); err != nil {
		t.Fatalf("the refreshed access token was rejected: %v", err)
	}
}

func TestResetClearsRevocationState(t *testing.T) {
	store, _ := newStore(t, 42)
	now := time.Now()
	login := Login{Username: "admin", Role: "admin"}

	token, _ := store.Issue(now, login, "access")
	store.LogOut()
	if _, err := store.Verify(now, token); err == nil {
		t.Fatal("the logout did not revoke a token issued before it")
	}

	// Reset is the clean slate a suite takes between tests, so a revocation
	// must not outlive it any more than a login does.
	store.Reset()
	if _, err := store.Verify(now, token); err != nil {
		t.Fatalf("a revocation survived the reset: %v", err)
	}

	// The record of what was issued goes too, or a logout in some later test
	// reaches back and revokes a token minted before the slate was wiped.
	store.LogOut()
	if _, err := store.Verify(now, token); err != nil {
		t.Fatalf("a logout after the reset revoked a token from before it: %v", err)
	}
}

func TestMagicTokensWorkExactlyOnce(t *testing.T) {
	store, _ := newStore(t, 42)

	token := store.IssueMagicToken("user")
	if got, ok := store.RedeemMagicToken(token); !ok || got != "user" {
		t.Fatalf("first redemption: %q %v", got, ok)
	}
	if _, ok := store.RedeemMagicToken(token); ok {
		t.Fatal("the token was redeemable twice")
	}
}

func TestPendingMagicLinksArePublishable(t *testing.T) {
	store, _ := newStore(t, 42)
	first := store.IssueMagicToken("user")
	store.IssueMagicToken("admin")

	pending := store.PendingMagicLinks()
	if len(pending) != 2 {
		t.Fatalf("%d pending links, want 2", len(pending))
	}

	store.RedeemMagicToken(first)
	if len(store.PendingMagicLinks()) != 1 {
		t.Fatal("a redeemed link is still listed as pending")
	}
}

func TestResetClearsEverything(t *testing.T) {
	store, _ := newStore(t, 42)
	now := time.Now()

	store.Attempt(now, "admin", "admin123")
	store.IssueMagicToken("user")
	store.Attempt(now, "admin", "wrong")
	store.Reset()

	if store.Current() != nil {
		t.Error("still logged in after reset")
	}
	if count, _ := store.Attempts(); count != 0 {
		t.Error("failures survived reset")
	}
	if len(store.PendingMagicLinks()) != 0 {
		t.Error("magic links survived reset")
	}
}

// The signing key is shared between sessions on the same seed by design, so a
// test can forge tokens. The CSRF token must not be: a guard that accepts a
// token minted in a different session is not guarding anything.
func TestCSRFTokensDifferBetweenSessions(t *testing.T) {
	store := session.NewStore(session.Options{Seed: 42})
	first := For(store.Open("worker-one"))
	second := For(store.Open("worker-two"))

	if first.SecretHex() != second.SecretHex() {
		t.Fatal("the signing key should be shared for a seed, so tokens can be forged deliberately")
	}
	if first.CSRF() == second.CSRF() {
		t.Fatal("two sessions share a CSRF token, which makes the guard cosmetic")
	}
}

func TestCSRFTokenIsStableWithinASession(t *testing.T) {
	store, _ := newStore(t, 42)

	if store.CSRF() != store.CSRF() {
		t.Fatal("the token changed between reads, which would make every form a race")
	}
}
