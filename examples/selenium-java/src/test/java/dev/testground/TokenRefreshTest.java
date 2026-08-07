package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.List;
import java.util.regex.Pattern;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.WebElement;

/** /app/token-refresh — an access token that dies mid-suite, and the difference between expired and rejected. */
class TokenRefreshTest extends Playground {

    private static final String PAGE = "/app/token-refresh";

    /** Comfortably past the sixty-second access lifetime, on the session clock rather than on the wall. */
    private static final long PAST_EXPIRY_MS = 61_000;

    @Test
    void aFreshTokenWorks() {
        open(PAGE);
        click("sign-in");
        click("call-api");

        waitForText("last-status", "200");
        assertTrue(text("identity").contains("user (user)"));
    }

    @Test
    void theTokenExpiresByMovingTheClockNotByWaitingSixtySeconds() {
        open(PAGE);
        click("sign-in");
        uncheck("auto-refresh");

        // Sixty seconds of real waiting would be sixty seconds every run, and it
        // would still land on a different test each time. The session clock puts
        // the expiry here, now, deterministically.
        advanceClock(PAST_EXPIRY_MS);
        click("call-api");

        waitForText("last-status", "401");
        waitForText("token-state", "expired");
    }

    @Test
    void expiredAndInvalidAreDifferent401sAndOnlyOneIsFixedByRefreshing() {
        open(PAGE);
        click("sign-in");

        // A token that was never valid.
        Object[] rejected = getWithBearer("/api/app/auth/me", "not.a.real.token");
        assertEquals(401L, rejected[0]);
        assertEquals("invalid", rejected[1]);

        // A token that was valid a minute ago. Same status code, and refreshing
        // fixes only this one -- a test that asserted on 401 alone would treat
        // the two as the same failure and reach for the wrong remedy.
        advanceClock(PAST_EXPIRY_MS);
        uncheck("auto-refresh");
        click("call-api");
        waitForText("last-reason", "expired");
    }

    @Test
    void a200AloneCannotTellARefreshFromANeverExpiredToken() {
        open(PAGE);
        click("sign-in");
        advanceClock(PAST_EXPIRY_MS);

        click("call-api");

        // The tempting assertion, and it is true of a page whose refresh path
        // never ran at all.
        waitForText("last-status", "200");

        // The counter is what turns it into evidence that the recovery happened.
        waitForText("refresh-count", "1");
        waitForPresent("identity");
    }

    @Test
    void theSigningKeyIsPublishedSoATokenCanBeBuiltOnPurpose() {
        // Not a secret, deliberately: a test that cannot sign cannot construct
        // the token it needs and is left waiting for one to go stale.
        open("/api/control/auth");

        assertTrue(
                Pattern.compile("\"signingKey\": \"[0-9a-f]{64}\"").matcher(driver.getPageSource()).find(),
                "the control plane stopped publishing a usable signing key");
    }

    @Test
    void aRefreshTokenIsNotAnAccessToken() {
        open(PAGE);
        click("sign-in");

        // Playwright asks for a second pair through its request context. The
        // equivalent here is a scripted login, which lands in the same pinned
        // session because the cookie goes with it.
        String refreshToken = (String) ((JavascriptExecutor) driver).executeAsyncScript(
                "const done = arguments[arguments.length - 1];"
                        + "fetch('/api/app/auth/login', {"
                        + "  method: 'POST',"
                        + "  headers: { 'content-type': 'application/json' },"
                        + "  body: JSON.stringify({ username: 'user', password: 'user123' })"
                        + "}).then(r => r.json()).then(b => done(b.refresh)).catch(e => done(String(e)));");

        Object[] wrongKind = getWithBearer("/api/app/auth/me", refreshToken);
        assertEquals(401L, wrongKind[0]);
        assertEquals(
                "wrong-kind",
                wrongKind[1],
                "a refresh token used as an access token is a third kind of 401 again");
    }

    /** Unticks a checkbox, and says nothing if it was already unticked. */
    private void uncheck(String id) {
        WebElement box = find(id);
        if (box.isSelected()) {
            box.click();
        }
    }

    /** Moves this session's clock forward, which is how expiry is made to happen on demand. */
    private void advanceClock(long ms) {
        driver.manage().timeouts().scriptTimeout(TIMEOUT);
        Object status = ((JavascriptExecutor) driver).executeAsyncScript(
                "const done = arguments[arguments.length - 1];"
                        + "fetch('/api/control/clock', {"
                        + "  method: 'POST',"
                        + "  headers: { 'content-type': 'application/json' },"
                        + "  body: JSON.stringify({ action: 'advance', ms: arguments[0] })"
                        + "}).then(r => done(r.status)).catch(e => done(String(e)));",
                ms);
        assertEquals(200L, status, "the control plane would not move the clock");
    }

    /**
     * A GET with a bearer token, returning its status and the reason it gave.
     *
     * <p>WebDriver cannot put a header on a navigation, and the whole lesson
     * here lives in the Authorization header, so the request is made from page
     * script. It still carries the pinned session cookie, so it is this test's
     * auth store being asked.
     */
    private Object[] getWithBearer(String path, String token) {
        driver.manage().timeouts().scriptTimeout(TIMEOUT);
        @SuppressWarnings("unchecked")
        List<Object> answer = (List<Object>) ((JavascriptExecutor) driver).executeAsyncScript(
                "const done = arguments[arguments.length - 1];"
                        + "fetch(arguments[0], { headers: { Authorization: 'Bearer ' + arguments[1] } })"
                        + "  .then(async r => done([r.status, (await r.json()).reason || '']))"
                        + "  .catch(e => done([0, String(e)]));",
                path, token);
        return answer.toArray();
    }
}
