package dev.testground;

import static java.nio.charset.StandardCharsets.UTF_8;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.net.URLEncoder;
import java.util.Map;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.JavascriptExecutor;

/** /classic/form-login — three refusals that look alike on screen and are three different statuses. */
class FormLoginTest extends Playground {

    private static final String PAGE = "/classic/form-login";

    /** The hidden token, as it appears in the served HTML of any session. */
    private static final Pattern CSRF_IN_HTML = Pattern.compile("name=\"csrf\" value=\"([a-f0-9]+)\"");

    /** A session belonging to nobody, used to prove a token cannot be borrowed. */
    private static final String NEIGHBOUR = "se-csrf-thief";

    @Test
    void aCorrectPasswordSignsYouIn() {
        signIn("admin", "admin123");

        assertTrue(text("welcome").contains("Ada Admin"));
        assertEquals("admin", text("current-role"));
    }

    @Test
    void theThreeRefusalsHaveThreeDifferentStatuses() {
        open(PAGE);
        String csrf = csrfToken();

        // Wrong password: the form was fine, the credentials were not.
        Map<String, Object> wrong = postForm(PAGE, form("csrf", csrf, "username", "admin", "password", "nope"));
        assertEquals(200L, wrong.get("status"));

        // Missing token: refused before the password was even considered, so
        // this says nothing at all about whether the password was right.
        Map<String, Object> noToken = postForm(PAGE, form("username", "admin", "password", "admin123"));
        assertEquals(403L, noToken.get("status"));

        // And here is why the status is the assertion. Both answers render the
        // same page with the same form and a populated login-error: a test that
        // reads the DOM has proved only that it is not signed in, which is true
        // of a credential bug, a token bug and its own earlier failures alike.
        assertTrue(html(wrong).contains("data-testid=\"login-form\""));
        assertTrue(html(noToken).contains("data-testid=\"login-form\""));
    }

    @Test
    void theCsrfTokenBelongsToThisSessionAndNoOther() {
        open(PAGE);

        // Playwright opens a second request context with its own session header.
        // WebDriver cannot put a header on a navigation at all -- which is why
        // this suite pins sessions with the cookie -- but a fetch from the page
        // can, so the second session is reachable without a second browser.
        //
        // It has to be an uncredentialled fetch, and this is the sharp edge of
        // doing it from inside the browser: the playground answers every request
        // with a Set-Cookie naming the session it resolved, so a credentialled
        // read of another session would repin this browser onto it and every
        // later request in the test would quietly be the thief's.
        //
        // The neighbour is reset first for the same reason this suite resets its
        // own session: the id is named rather than randomised, so it outlives
        // the run, and a neighbour still signed in from an earlier one serves no
        // form and no token at all.
        http("POST", "/api/control/reset", null, null, NEIGHBOUR);
        String foreign = csrfFrom(html(getAs(PAGE, NEIGHBOUR)));
        assertNotEquals(csrfToken(), foreign, "the two sessions minted the same token, so nothing is being guarded");

        Map<String, Object> response = postForm(PAGE, form("csrf", foreign, "username", "admin", "password", "admin123"));
        assertEquals(403L, response.get("status"), "a token from another session is not a token");
    }

    @Test
    void fiveWrongPasswordsThrottleTheSixthHoweverRightItIs() {
        open(PAGE);
        String csrf = csrfToken();
        failFiveTimes(csrf);

        Map<String, Object> correct = postForm(PAGE, form("csrf", csrf, "username", "admin", "password", "admin123"));
        assertEquals(429L, correct.get("status"), "the throttle is about the attempts, not about the credentials");
    }

    @Test
    void theThrottleLiftsByMovingTheClockRatherThanByWaiting() {
        open(PAGE);
        String csrf = csrfToken();
        failFiveTimes(csrf);

        // Thirty seconds of real waiting would be thirty seconds every run. The
        // lockout is measured on the session clock, so the control plane moves
        // past it instantly and only for this session.
        postJson("/api/control/clock", "{\"action\":\"advance\",\"ms\":40000}");

        Map<String, Object> after = postForm(PAGE, form("csrf", csrf, "username", "admin", "password", "admin123"));

        // Playwright stops on the 303 with maxRedirects: 0. A fetch cannot be
        // told to do that -- redirect: 'manual' yields an opaque response with
        // no status on it -- so the evidence that the post was accepted is that
        // it was redirected onto a page rather than answered in place, which is
        // exactly what a 303 means.
        assertEquals(200L, after.get("status"));
        assertEquals(Boolean.TRUE, after.get("redirected"));

        driver.navigate().refresh();
        waitForPresent("welcome");
    }

    @Test
    void aSuiteThatFailsLoginsOnPurposeHasToCleanUpAfterItself() {
        open(PAGE);
        failFiveTimes(csrfToken());

        // Without this the next test in this session meets a throttle it did not
        // cause and cannot explain -- and because the session id here is derived
        // from the class name rather than randomised, it would outlive the run
        // as well.
        postJson("/api/control/auth/reset", null);

        signIn("admin", "admin123");
        waitForPresent("welcome");
    }

    @Test
    void rememberMeIsRecordedOnTheLogin() {
        open(PAGE);
        find("field-username").sendKeys("user");
        find("field-password").sendKeys("user123");
        click("field-remember");
        click("submit");

        waitForPresent("remembered");
    }

    @Test
    void loggingOutEndsTheLoginOnTheServer() {
        signIn("admin", "admin123");
        click("logout");

        waitForPresent("login-form");

        // The browser losing sight of the login is not the same as the login
        // ending, and only one of those is worth asserting. The server is the
        // authority, so ask it.
        assertNull(json("/api/control/auth").get("login"));
    }

    /** Signs in through the form, the way a user would. */
    private void signIn(String username, String password) {
        open(PAGE);
        find("field-username").sendKeys(username);
        find("field-password").sendKeys(password);
        click("submit");
    }

    /** The token this session's form is carrying. Read, never invented. */
    private String csrfToken() {
        return find("csrf-token").getDomProperty("value");
    }

    /** Pulls the token out of served HTML, for the copy belonging to another session. */
    private String csrfFrom(String page) {
        Matcher matched = CSRF_IN_HTML.matcher(page);
        assertTrue(matched.find(), "no CSRF token in that page");
        return matched.group(1);
    }

    /** Spends the session's five attempts, which is what arms the throttle. */
    private void failFiveTimes(String csrf) {
        for (int attempt = 0; attempt < 5; attempt++) {
            postForm(PAGE, form("csrf", csrf, "username", "admin", "password", "nope"));
        }
        driver.navigate().refresh();
        assertEquals("5", text("attempts"));
    }

    /** Builds a form body from alternating names and values. */
    private static String form(String... pairs) {
        StringBuilder body = new StringBuilder();
        for (int i = 0; i < pairs.length; i += 2) {
            if (i > 0) {
                body.append('&');
            }
            body.append(URLEncoder.encode(pairs[i], UTF_8)).append('=').append(URLEncoder.encode(pairs[i + 1], UTF_8));
        }
        return body.toString();
    }

    private static String html(Map<String, Object> response) {
        return String.valueOf(response.get("text"));
    }

    private Map<String, Object> postForm(String path, String body) {
        return http("POST", path, "application/x-www-form-urlencoded", body, null);
    }

    private Map<String, Object> postJson(String path, String body) {
        return http("POST", path, "application/json", body, null);
    }

    /** A GET issued as some other session, which only a header can ask for. */
    private Map<String, Object> getAs(String path, String session) {
        return http("GET", path, null, null, session);
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

    /**
     * Issues a request from the page and reports the status, whether it was
     * redirected, and the body.
     *
     * <p>WebDriver has no request API and no response object, so every status
     * this class asserts on comes from here. Being same-origin it carries the
     * pinned session cookie by default, and unlike a navigation it can override
     * that with a header -- which is what makes a second session reachable
     * without a second browser.
     *
     * <p>Naming a session switches the fetch to uncredentialled, so the answer's
     * Set-Cookie is neither sent nor stored. Without that, reading another
     * session repins the browser onto it.
     */
    @SuppressWarnings("unchecked")
    private Map<String, Object> http(String method, String path, String contentType, String body, String session) {
        driver.manage().timeouts().scriptTimeout(TIMEOUT);
        return (Map<String, Object>) ((JavascriptExecutor) driver).executeAsyncScript(
                "const [method, path, contentType, body, session, done] = arguments;"
                        + "const init = { method, headers: {} };"
                        + "if (contentType !== null) { init.headers['content-type'] = contentType; }"
                        + "if (session !== null) {"
                        + "  init.headers['X-Playground-Session'] = session;"
                        + "  init.credentials = 'omit';"
                        + "}"
                        + "if (body !== null) { init.body = body; }"
                        + "fetch(path, init)"
                        + "  .then(async r => done({ status: r.status, redirected: r.redirected, text: await r.text() }))"
                        + "  .catch(e => done({ status: -1, redirected: false, text: String(e) }));",
                method, path, contentType, body, session);
    }
}
