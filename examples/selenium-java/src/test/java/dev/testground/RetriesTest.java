package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.Cookie;
import org.openqa.selenium.JavascriptExecutor;

/** /app/retries — an endpoint that refuses its first calls, and what retrying does to an assertion. */
class RetriesTest extends Playground {

    private static final String PAGE = "/app/retries";

    @Test
    void retryingEventuallySucceedsWhereAskingOnceDoesNot() {
        open(PAGE + "?failFirst=2");

        click("fetch-once");
        waitForText("outcome", "failed");
        waitForText("attempt-count", "1");

        click("reset-endpoint");
        click("fetch-retrying");

        // Same server, same endpoint, opposite result. Which button was pressed
        // is the entire difference, so a test that asserts on the outcome
        // without saying which one it pressed is asserting on the retry policy
        // and calling it a feature test.
        waitForText("outcome", "succeeded");
        waitForText("attempt-count", "3");
    }

    @Test
    void theOutcomeAloneCannotTellThreeAttemptsFromOne() {
        open(PAGE + "?failFirst=0");
        click("fetch-retrying");

        // Identical to the assertion that ended the test above, and a completely
        // different story behind it. This is why retries hide breakage: a suite
        // that only looks here passes against a service failing two calls in
        // three and reports nothing.
        waitForText("outcome", "succeeded");

        waitForText("attempt-count", "1");
    }

    @Test
    void everyAttemptIsRecordedWithTheStatusItGot() {
        open(PAGE + "?failFirst=2");
        click("fetch-retrying");
        waitForText("outcome", "succeeded");

        assertEquals(3, count("attempt-row"));
        assertEquals(2, countWithStatus(503), "two refusals were expected before the answer");
        assertEquals(1, countWithStatus(200));
    }

    @Test
    void aFixedAttemptBudgetCannotRideOutAnEndpointThatNeverRecovers() {
        open(PAGE + "?failFirst=0");

        // The control plane refuses every call the endpoint would have answered,
        // which is the failure a retry loop cannot distinguish from slowness.
        // It lands in this test's session only, because the pinned cookie rides
        // on the request the page makes for us.
        flakeEveryCall();

        click("fetch-retrying");

        waitForText("outcome", "failed");
        waitForText("attempt-count", "6");
        assertEquals(6, countWithStatus(503));
        assertEquals(0, count("payload"), "nothing succeeded, so there is no body to show");
    }

    @Test
    void theRefusalCounterIsPerSession() {
        open(PAGE + "?failFirst=1");
        click("fetch-retrying");
        waitForText("outcome", "succeeded");

        // Playwright opens a second request context with a different session
        // header. WebDriver has no request context, so the equivalent is to move
        // this browser to a neighbouring session by swapping the cookie and ask
        // the endpoint directly.
        //
        // The neighbour is reset first on purpose: unlike a Playwright request
        // context it is not thrown away at the end of the run, so without this
        // the second time anyone runs this class the neighbour would already
        // have spent its one refusal.
        driver.manage().addCookie(new Cookie("playground_session", "se-retries-other-worker", "/"));
        resetSession();

        assertEquals(
                503L,
                statusOf("/api/app/retries/data?failFirst=1"),
                "a neighbouring session met an endpoint this test had already spent");
    }

    /**
     * How many attempt rows came back with a given status.
     *
     * <p>Narrowing a published test id by the attribute beside it is still
     * locating by the contract; it is the row's identity, not its position.
     */
    private int countWithStatus(int status) {
        return driver.findElements(By.cssSelector("[data-testid='attempt-row'][data-status='" + status + "']")).size();
    }

    /** Refuses every call to this challenge, for this session only. */
    private void flakeEveryCall() {
        driver.manage().timeouts().scriptTimeout(TIMEOUT);
        Object status = ((JavascriptExecutor) driver).executeAsyncScript(
                "const done = arguments[arguments.length - 1];"
                        + "fetch('/api/control/flake', {"
                        + "  method: 'POST',"
                        + "  headers: { 'content-type': 'application/json' },"
                        + "  body: JSON.stringify({ challenge: 'retries', probability: 1 })"
                        + "}).then(r => done(r.status)).catch(e => done(String(e)));");
        assertEquals(200L, status, "the control plane refused to install the flake rule");
    }

    /** The status code a plain GET gets, read from page script because WebDriver never sees one. */
    private Object statusOf(String path) {
        driver.manage().timeouts().scriptTimeout(TIMEOUT);
        return ((JavascriptExecutor) driver).executeAsyncScript(
                "const done = arguments[arguments.length - 1];"
                        + "fetch(arguments[0]).then(r => done(r.status)).catch(e => done(String(e)));",
                path);
    }
}
