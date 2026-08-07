package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.time.Duration;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.TimeoutException;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.support.ui.ExpectedCondition;
import org.openqa.selenium.support.ui.ExpectedConditions;
import org.openqa.selenium.support.ui.WebDriverWait;

/** /legacy/history — pushState, replaceState and a back button that reconstructs the page from the URL. */
class HistoryTest extends Playground {

    private static final String PAGE = "/legacy/history";

    @Test
    void pushStateChangesTheUrlWithoutFetchingAnything() {
        open(PAGE);
        WebElement step = find("current-step");
        markDocument();

        click("push-one");

        waitForUrlEndingWith("/legacy/history?step=1");
        waitForText("current-step", "1");

        // The Playwright suite watches the request list and sees that no
        // document was asked for. WebDriver has no network events at all, so
        // the proof has to come from inside the page: a value set on window
        // before the click is still there afterwards, and a real document fetch
        // would have wiped it.
        assertTrue(documentSurvived(), "the page was re-fetched, so this was a navigation after all");

        // The trap in the shape a Selenium user meets it. stalenessOf is the
        // habitual way to wait for a page to change, and here it waits for a
        // change that is never coming: the element was never replaced, only its
        // text was. The wait is short on purpose -- the point is that it expires.
        assertThrows(
                TimeoutException.class,
                () -> new WebDriverWait(driver, Duration.ofSeconds(2))
                        .until(ExpectedConditions.stalenessOf(step)),
                "the element went stale, so something did reload the document");
    }

    @Test
    void backRebuildsThePageFromTheUrlAlone() {
        open(PAGE);
        click("push-one");
        click("push-two");
        waitForText("current-step", "2");

        driver.navigate().back();

        // Both halves matter. The server never served this URL, so what is on
        // screen was reconstructed by script from the address bar -- and either
        // half can be right on its own while the pair disagrees.
        waitForUrlEndingWith("/legacy/history?step=1");
        waitForText("current-step", "1");
        waitForText("popstate-count", "1");
    }

    @Test
    void replaceStateLeavesNoEntrySoBackSkipsPastIt() {
        open(PAGE);
        click("push-one");
        click("replace");
        waitForText("current-step", "replaced");

        driver.navigate().back();

        // Not back to step 1: replaceState overwrote that entry rather than
        // adding one, so a single back goes one step further than it looks.
        waitForUrlEndingWith("/legacy/history");
        waitForText("current-step", "none");
    }

    @Test
    void aHashChangeNeverReachesTheServer() {
        open(PAGE);
        markDocument();

        click("hash-link");

        waitForText("current-hash", "#section-two");
        waitForUrlEndingWith("/legacy/history#section-two");
        assertTrue(documentSurvived(), "the fragment was sent to the server, which cannot happen");

        // The fragment link is still a same-document navigation, so Chrome
        // fires popstate for it and the rebuild counter moves exactly as it
        // does for the back button. A test using that counter to tell "the
        // user went back" from "the user followed a link" cannot: the counter
        // says the page rebuilt itself, not why.
        assertEquals("1", text("popstate-count"));
    }

    /**
     * Stamps the current document so a later check can tell whether it is the
     * same one. There is no WebDriver equivalent of Playwright's request events,
     * and this answers the question those events are being used to ask.
     */
    private void markDocument() {
        ((JavascriptExecutor) driver).executeScript("window.__sameDocument = true;");
    }

    private boolean documentSurvived() {
        return Boolean.TRUE.equals(
                ((JavascriptExecutor) driver).executeScript("return window.__sameDocument === true;"));
    }

    /** Waits for the address bar to settle on a URL, which lags the click by a frame. */
    private void waitForUrlEndingWith(String suffix) {
        wait.until((ExpectedCondition<Boolean>) d -> d.getCurrentUrl().endsWith(suffix));
    }
}
