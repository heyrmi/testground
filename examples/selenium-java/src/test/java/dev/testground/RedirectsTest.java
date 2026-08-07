package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.Map;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.JavascriptExecutor;

/** /classic/redirects — the URL you asked for and the page you are reading are two strings. */
class RedirectsTest extends Playground {

    /** The method the destination page says it received. */
    private static final Pattern METHOD = Pattern.compile("landed-method\">([A-Z]+)");

    @Test
    void theBrowserFollowsTheWholeChainAndLandsSomewhereElse() {
        open("/classic/redirects");
        click("chain-link");

        waitForPresent("landed");
        assertTrue(
                driver.getCurrentUrl().endsWith("/classic/redirects/landed?via=chain"),
                "asserting on the requested URL rather than the arrived-at one: " + driver.getCurrentUrl());
        assertEquals("3", text("landed-hops"));
    }

    @Test
    void theHopCountComesFromTheBrowserBecauseTheDriverKeepsNoRequests() {
        open("/classic/redirects/chain/1");

        // The Playwright suite walks response.request().redirectedFrom() back up
        // the chain. WebDriver models no requests at all, so there is nothing to
        // walk -- but the browser counted the hops while it followed them, and
        // Navigation Timing hands the count to page script. Same fact, and the
        // only route to it from here.
        assertEquals(3L, navigationTiming("redirectCount"));
        assertEquals(200L, navigationTiming("responseStatus"));
        assertTrue(driver.getCurrentUrl().contains("via=chain"));
    }

    @Test
    void threeOhSevenAndThreeOhEightKeepTheMethodTheOthersAreFreeToRewriteIt() {
        open("/classic/redirects");

        assertEquals("GET", methodAfter(301));
        assertEquals("GET", methodAfter(302));
        assertEquals("GET", methodAfter(303));
        assertEquals("POST", methodAfter(307), "a POST through 307 stays a POST");
        assertEquals("POST", methodAfter(308), "a POST through 308 stays a POST");
    }

    @Test
    void metaRefreshIsNotARedirect() {
        open("/classic/redirects");

        // No Location header and no redirect status: the document that answered
        // is the document that was asked for, and nothing was followed.
        Map<String, Object> response = fetch("GET", "/classic/redirects/meta", null);
        assertEquals(200L, response.get("status"));
        assertEquals(Boolean.FALSE, response.get("redirected"));
    }

    @Test
    void waitingForTheNavigationLandsYouOnThePageYouWereLeaving() {
        open("/classic/redirects/meta");

        // The approach that looks right and is not. get() waits for the load
        // event, the load event fired, and the document it fired on is the one
        // about to replace itself a second later. A test that treats "the
        // navigation finished" as "I am on the destination" is on the wrong page
        // and will say so in a way that reads like a flake.
        assertTrue(driver.getCurrentUrl().endsWith("/classic/redirects/meta"));
        assertEquals(1, count("meta-waiting"));
        assertEquals(0, count("landed"));

        // Waiting for something only the destination has is what actually works.
        waitForPresent("landed");
        assertEquals("meta", text("landed-via"));
    }

    /** Posts through one redirect code and reports the method the destination received. */
    private String methodAfter(int code) {
        Map<String, Object> response = fetch("POST", "/classic/redirects/code/" + code, "x=1");
        assertEquals(200L, response.get("status"), "the " + code + " chain did not land");

        Matcher matched = METHOD.matcher(String.valueOf(response.get("text")));
        assertTrue(matched.find(), "the " + code + " destination did not report a method");
        return matched.group(1);
    }

    /** One field of the navigation currently on screen, as the browser recorded it. */
    private long navigationTiming(String field) {
        Object value = ((JavascriptExecutor) driver).executeScript(
                "const entry = performance.getEntriesByType('navigation')[0];"
                        + "return entry ? entry[arguments[0]] : null;",
                field);
        assertNotNull(value, "the browser recorded no " + field + " for this navigation");
        return (Long) value;
    }

    /**
     * Issues a request from the page and reports what came back.
     *
     * <p>A browser navigation cannot POST, and WebDriver cannot issue a request
     * of its own, so the only way to send a method other than GET is from inside
     * the page. Being same-origin, it carries this class's session cookie.
     *
     * <p>What it cannot do is stop on the redirect the way Playwright's
     * {@code maxRedirects: 0} does: fetch with {@code redirect: 'manual'} yields
     * an opaque response with no status at all. So the evidence that 307 kept the
     * method is the method the destination reports, which is the better evidence
     * anyway -- it is the thing the redirect code was supposed to affect.
     */
    @SuppressWarnings("unchecked")
    private Map<String, Object> fetch(String method, String path, String body) {
        driver.manage().timeouts().scriptTimeout(TIMEOUT);
        return (Map<String, Object>) ((JavascriptExecutor) driver).executeAsyncScript(
                "const [method, path, body, done] = arguments;"
                        + "const init = { method };"
                        + "if (body !== null) {"
                        + "  init.headers = { 'content-type': 'application/x-www-form-urlencoded' };"
                        + "  init.body = body;"
                        + "}"
                        + "fetch(path, init)"
                        + "  .then(async r => done({"
                        + "    status: r.status, url: r.url, redirected: r.redirected, text: await r.text() }))"
                        + "  .catch(e => done({ status: -1, url: '', redirected: false, text: String(e) }));",
                method, path, body);
    }
}
