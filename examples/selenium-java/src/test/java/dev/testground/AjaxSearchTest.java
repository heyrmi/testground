package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import java.util.List;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.StaleElementReferenceException;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.support.ui.ExpectedCondition;

/** /legacy/ajax-search — a debounced search that replaces its whole results list on every response. */
class AjaxSearchTest extends Playground {

    private static final String PAGE = "/legacy/ajax-search";

    @Test
    void typingEightCharactersDoesNotSendEightRequests() {
        open(PAGE);
        find("search-input").sendKeys("kaminski");

        // sendKeys types one character at a time, so the page saw eight input
        // events a few milliseconds apart and cancelled the pending timer each
        // time. Waiting for the results to settle and only then reading the
        // counter is what proves the debounce collapsed them: asserting the
        // counter first would pass against a page that had sent nothing yet.
        waitForText("search-count", "20");
        assertEquals("1", text("request-count"), "the debounce did not collapse the burst");
    }

    @Test
    void assertingBeforeTheDebounceMeasuresNothing() {
        open(PAGE);
        find("search-input").sendKeys("zof");

        // The trap: typing has finished, so the page looks ready. It is not --
        // the 300 ms timer is still running and no request has left the browser.
        // A test that reads the results here reads the state before its search.
        assertEquals("0", text("request-count"));

        // Waiting for the state rather than for a duration is the whole fix.
        // No sleep of 300 ms, which would be both slower and wrong on a loaded
        // machine.
        waitForText("search-count", "16");
    }

    @Test
    void theTotalIsNotTheNumberOfRowsOnScreen() {
        open(PAGE);
        find("search-input").sendKeys("an");

        // Two different numbers, and a test that reads the wrong one cannot
        // tell a search returning 88 matches from one returning 25.
        waitForText("search-count", "88");
        assertEquals("25", text("search-shown"));
        assertEquals(25, count("search-row"));
    }

    @Test
    void rowsAreDetachedByTheNextSearchRatherThanUpdated() {
        open(PAGE);
        WebElement input = find("search-input");
        input.sendKeys("zof");
        waitForText("search-count", "16");

        WebElement held = find("search-row");

        input.clear();
        input.sendKeys("ova");
        waitForTextContaining("search-empty", "No names match");

        // The Playwright suite asks the held handle whether it is still
        // connected. Selenium has no such question, and does not need one: the
        // reference itself has gone bad, and touching it throws. Same lesson,
        // reported by the tool instead of by an assertion -- which is why rows
        // have to be re-located after every search rather than held across one.
        assertThrows(
                StaleElementReferenceException.class,
                held::getText,
                "the row survived a search, so the list was updated rather than replaced");
    }

    @Test
    void aQueryMatchingNothingSaysSo() {
        open(PAGE);
        find("search-input").sendKeys("ova");

        // Not waitForText: the empty marker starts out saying something else
        // entirely, so matching on a substring is what distinguishes "nothing
        // searched for yet" from "searched, and nothing matched".
        waitForTextContaining("search-empty", "No names match \"ova\"");
        assertEquals("0", text("search-count"));
    }

    /**
     * Waits for an element's text to contain a fragment.
     *
     * <p>The base class offers equality, which is the right default. This page
     * needs the looser check because the marker it reuses carries different
     * wording before and after the first search.
     */
    private void waitForTextContaining(String id, String fragment) {
        wait.until((ExpectedCondition<Boolean>) d -> {
            List<WebElement> found = d.findElements(testId(id));
            return !found.isEmpty() && found.get(0).getText().contains(fragment);
        });
    }
}
