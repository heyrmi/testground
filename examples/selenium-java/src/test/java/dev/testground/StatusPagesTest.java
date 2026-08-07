package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.Map;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.JavascriptExecutor;

/** /classic/status-pages — a page that rendered perfectly can still have failed. */
class StatusPagesTest extends Playground {

    /** Every code the challenge serves, in the order the page lists them. */
    private static final int[] SERVED = {400, 401, 403, 404, 418, 429, 500, 502, 503, 504};

    @Test
    void aRenderedErrorPageIsStillAnError() {
        open("/classic/status-pages/500");

        // Every content assertion passes. The heading is right, the reason is
        // there, the styling is intact.
        assertEquals("500", text("status-code"));
        waitForPresent("status-reason");

        // And the response was a 500 all the same.
        assertEquals(500L, navigationStatus());
    }

    @Test
    void aNavigationThatFailedLooksLikeOneThatSucceeded() {
        open("/classic/status-pages");
        assertTrue(driver.getCurrentUrl().endsWith("/classic/status-pages"));
        assertTrue(driver.getPageSource().contains("</html>"));

        open("/classic/status-pages/500");

        // The trap, and it is a trap the driver walks you into rather than one
        // the page sets. Both observations still hold: the URL is the one that
        // was asked for and a complete document parsed. WebDriver has nothing
        // to contradict them with -- get() returns void, and there is no
        // response object anywhere in the API -- so a suite that checks only
        // this has just asserted that a 500 is healthy.
        assertTrue(driver.getCurrentUrl().endsWith("/classic/status-pages/500"));
        assertTrue(driver.getPageSource().contains("</html>"));

        // Asking the browser rather than the driver is what separates them.
        assertEquals(500L, navigationStatus());
    }

    @Test
    void eachCodeIsServedWithItsOwnPage() {
        for (int code : SERVED) {
            open("/classic/status-pages/" + code);

            assertEquals(String.valueOf(code), text("status-code"), "the page for " + code);
            assertEquals((long) code, navigationStatus(), "the response for " + code);
        }
    }

    @Test
    void theThrottlingCodesSayHowLongToWait() {
        open("/classic/status-pages");

        // Retry-After is a response header, and a header is the one thing a
        // navigation leaves no trace of: Navigation Timing records the status
        // but not the headers. A same-origin fetch from the page carries the
        // session cookie and can read every header on the answer, so this is
        // where the second back channel earns its place.
        Map<String, Object> throttled = fetch("/classic/status-pages/429");
        assertEquals(429L, throttled.get("status"));
        assertEquals("5", throttled.get("retryAfter"));

        Map<String, Object> unavailable = fetch("/classic/status-pages/503");
        assertEquals(503L, unavailable.get("status"));
        assertEquals("10", unavailable.get("retryAfter"));

        // The page prints it too, so a test can honour the wait it was given
        // without leaving the DOM. Inventing a backoff instead is ignoring the
        // only reliable answer the server offered.
        open("/classic/status-pages/503");
        assertEquals("10", text("status-retry-after"));
    }

    @Test
    void aCodeThisPageDoesNotServeIsACleanNotFound() {
        open("/classic/status-pages/599");

        assertEquals(404L, navigationStatus());
    }

    /**
     * The HTTP status of the navigation currently on screen.
     *
     * <p>Not a translation of Playwright's {@code response.status()}, because
     * WebDriver has no response to translate: navigation returns void and the
     * protocol never reports what the server answered. The browser did record
     * it, though, and PerformanceNavigationTiming publishes it to page script,
     * so the test asks the page instead of the driver.
     */
    private long navigationStatus() {
        Object status = ((JavascriptExecutor) driver).executeScript(
                "const entry = performance.getEntriesByType('navigation')[0];"
                        + "return entry ? entry.responseStatus : null;");
        assertNotNull(status, "the browser did not report a status for this navigation");
        return (Long) status;
    }

    /**
     * Fetches a playground path from the page and reports the status and the
     * Retry-After header.
     *
     * <p>Same-origin, so it carries this class's session cookie and lands in
     * the same session the navigations do, and no header is hidden from it the
     * way a cross-origin read would hide one.
     */
    @SuppressWarnings("unchecked")
    private Map<String, Object> fetch(String path) {
        driver.manage().timeouts().scriptTimeout(TIMEOUT);
        return (Map<String, Object>) ((JavascriptExecutor) driver).executeAsyncScript(
                "const [path, done] = arguments;"
                        + "fetch(path)"
                        + "  .then(r => done({ status: r.status, retryAfter: r.headers.get('retry-after') }))"
                        + "  .catch(e => done({ status: -1, retryAfter: String(e) }));",
                path);
    }
}
