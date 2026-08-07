package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.io.ByteArrayOutputStream;
import java.nio.ByteBuffer;
import java.util.Map;

import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.WebElement;

/** /classic/two-factor — two flows that cannot be finished from the page, and the back channel that finishes them. */
class TwoFactorTest extends Playground {

    private static final String PAGE = "/classic/two-factor";

    /** How long a TOTP code is valid for, in seconds. */
    private static final long PERIOD = 30;

    @Test
    void thePasswordAloneLeavesTheLoginHalfDone() {
        passwordStep();

        // Half a login is neither a failed one nor a successful one, and a suite
        // that only asks "am I signed in" reads it as the failure it is not.
        waitForPresent("pending-notice");
        waitForPresent("code-form");
        assertEquals(0, count("welcome"));
    }

    @Test
    void theCodeComesFromTheBackChannelBecauseItCannotBeAFixture() {
        passwordStep();

        // A code recorded into a fixture is wrong within thirty seconds of being
        // recorded. The control plane publishes the one valid on this session's
        // clock at the moment it is asked, which is the only form of this that
        // still works tomorrow.
        enterCode(String.valueOf(json("/api/control/auth").get("totpCode")));

        assertTrue(text("welcome").contains("Tam Two-Factor"));
    }

    @Test
    void aTestCanComputeTheCodeItselfFromThePublishedSecret() {
        passwordStep();
        String secret = String.valueOf(json("/api/control/auth").get("totpSecret"));

        // The Playwright suite computes this in the page with WebCrypto, because
        // that is where its code already runs. Java has HMAC in the standard
        // library, so this one does it in the test process instead -- which is
        // the better place for it: an implementation proved here is one a real
        // suite could ship, rather than one that only exists inside a browser it
        // happens to be driving.
        enterCode(totp(secret, System.currentTimeMillis() / 1000));

        waitForPresent("welcome");
    }

    @Test
    void aStaleCodeIsRefused() {
        passwordStep();
        String code = String.valueOf(json("/api/control/auth").get("totpCode"));

        // Two minutes past the window it was minted for. Moving the clock rather
        // than sleeping is what makes an expiry test deterministic instead of a
        // race against thirty seconds.
        postJson("/api/control/clock", "{\"action\":\"advance\",\"ms\":120000}");

        enterCode(code);

        assertTrue(text("login-error").contains("not valid at this moment"));
        assertEquals(0, count("welcome"));
    }

    @Test
    void theCodeFollowsTheClockSoAMovedClockHasItsOwnValidCode() {
        passwordStep();
        postJson("/api/control/clock", "{\"action\":\"advance\",\"ms\":120000}");

        // The same secret and the same account at a different instant: asked
        // after the move, the back channel answers with the code valid there.
        enterCode(String.valueOf(json("/api/control/auth").get("totpCode")));

        waitForPresent("welcome");
    }

    @Test
    void aSignInLinkIsRetrievableWithoutAnInbox() {
        open(PAGE);
        click("send-magic-link");
        waitForPresent("magic-link");

        // The page lists the link, because the playground has nowhere to send
        // it. Reading it from the control plane instead is the shape that
        // survives contact with a real system, where the link is in an inbox no
        // browser test can open and the page shows nothing at all.
        Map<String, Object> links = magicLinks();
        assertEquals(1, links.size());

        String token = links.keySet().iterator().next();
        assertEquals("user", links.get(token));

        open(PAGE + "/magic/" + token);
        assertTrue(text("welcome").contains("Uma User"));
    }

    @Test
    void aLinkWorksExactlyOnce() {
        open(PAGE);
        click("send-magic-link");
        waitForPresent("magic-link");
        String token = magicLinks().keySet().iterator().next();

        open(PAGE + "/magic/" + token);
        waitForPresent("welcome");

        // Cleared first, so the second attempt is refused for being spent rather
        // than waved through because somebody is already signed in.
        postJson("/api/control/auth/reset", null);

        // And read as a status rather than as a page. Navigating there would
        // render the refusal without ever telling WebDriver it was a 404, which
        // is the same blind spot the status-pages challenge is about.
        assertEquals(404L, get(PAGE + "/magic/" + token).get("status"), "a single-use token was accepted twice");
    }

    /** Gets past the password half of the login, which is all this account's password buys. */
    private void passwordStep() {
        open(PAGE);
        // Cleared first: this page pre-fills the username, so sendKeys alone
        // would append to it and post "twofactortwofactor".
        find("field-username").clear();
        find("field-username").sendKeys("twofactor");
        find("field-password").sendKeys("twofactor123");
        click("submit");
    }

    /**
     * Types a code and submits the form it belongs to.
     *
     * <p>Two more obvious ways both failed on CI and neither failed here, which
     * is the useful part. Clicking the button is delivered to a point, so it
     * depends on where the layout puts it, and Linux resolves different fonts
     * than macOS: the click was reported as successful and no request was made.
     * Enter in the field depends instead on the field holding focus at the
     * instant the key is dispatched, and the server's request log shows three
     * runs in four where that produced no request either.
     *
     * <p>Submitting the form has neither dependency. On a page with no script
     * this is exactly what pressing the button does -- there is no submit
     * handler to bypass and nothing to validate -- so the challenge is still
     * being exercised rather than stepped around. Clicking a button for its own
     * sake belongs in ButtonsTest, where that is the subject.
     */
    private void enterCode(String code) {
        WebElement field = find("field-code");
        field.sendKeys(code);

        // Guarded rather than assumed: if the keystrokes never landed, the
        // failure should say so here instead of looking like a refused code
        // four assertions later.
        assertEquals(
                code,
                field.getDomProperty("value"),
                "the code never reached the field, so the submission below proves nothing");

        find("code-form").submit();
    }

    @SuppressWarnings("unchecked")
    private Map<String, Object> magicLinks() {
        return (Map<String, Object>) json("/api/control/auth").get("magicLinks");
    }

    /**
     * The six-digit code for a base32 secret at an instant, computed here rather
     * than asked for.
     *
     * <p>RFC 6238 with the parameters the playground publishes: HMAC-SHA1,
     * thirty-second periods, six digits. The server allows one period of skew
     * either way, so a code computed a moment before it is typed still lands.
     */
    private static String totp(String base32Secret, long epochSeconds) {
        try {
            Mac mac = Mac.getInstance("HmacSHA1");
            mac.init(new SecretKeySpec(decodeBase32(base32Secret), "HmacSHA1"));
            byte[] hash = mac.doFinal(ByteBuffer.allocate(8).putLong(epochSeconds / PERIOD).array());

            int offset = hash[hash.length - 1] & 0x0f;
            int binary = ((hash[offset] & 0x7f) << 24)
                    | ((hash[offset + 1] & 0xff) << 16)
                    | ((hash[offset + 2] & 0xff) << 8)
                    | (hash[offset + 3] & 0xff);
            return String.format("%06d", binary % 1_000_000);
        } catch (Exception e) {
            throw new IllegalStateException("could not compute the code", e);
        }
    }

    /** Base32 without padding, which the JDK ships no decoder for. */
    private static byte[] decodeBase32(String secret) {
        String alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";
        ByteArrayOutputStream out = new ByteArrayOutputStream();

        int buffer = 0;
        int bits = 0;
        for (char letter : secret.toCharArray()) {
            buffer = (buffer << 5) | alphabet.indexOf(letter);
            bits += 5;
            if (bits >= 8) {
                out.write((buffer >>> (bits - 8)) & 0xff);
                bits -= 8;
            }
        }
        return out.toByteArray();
    }

    /** A control-plane document, parsed by the browser rather than by hand. */
    @SuppressWarnings("unchecked")
    private Map<String, Object> json(String path) {
        driver.manage().timeouts().scriptTimeout(TIMEOUT);
        return (Map<String, Object>) ((JavascriptExecutor) driver).executeAsyncScript(
                "const [path, done] = arguments;"
                        + "fetch(path).then(r => r.json()).then(done).catch(e => done({ error: String(e) }));",
                path);
    }

    private Map<String, Object> get(String path) {
        return http("GET", path, null, null);
    }

    private Map<String, Object> postJson(String path, String body) {
        return http("POST", path, "application/json", body);
    }

    /**
     * Issues a request from the page and reports the status.
     *
     * <p>WebDriver has no response object and cannot POST at all, so both the
     * statuses and the control-plane commands in this class go through here.
     * Same-origin, so the pinned session cookie travels with them.
     */
    @SuppressWarnings("unchecked")
    private Map<String, Object> http(String method, String path, String contentType, String body) {
        driver.manage().timeouts().scriptTimeout(TIMEOUT);
        return (Map<String, Object>) ((JavascriptExecutor) driver).executeAsyncScript(
                "const [method, path, contentType, body, done] = arguments;"
                        + "const init = { method, headers: {} };"
                        + "if (contentType !== null) { init.headers['content-type'] = contentType; }"
                        + "if (body !== null) { init.body = body; }"
                        + "fetch(path, init)"
                        + "  .then(async r => done({ status: r.status, text: await r.text() }))"
                        + "  .catch(e => done({ status: -1, text: String(e) }));",
                method, path, contentType, body);
    }
}
