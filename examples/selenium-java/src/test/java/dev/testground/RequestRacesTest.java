package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.time.Duration;
import java.util.List;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.support.ui.ExpectedCondition;
import org.openqa.selenium.support.ui.WebDriverWait;

/** /app/request-races — the older answer arriving last, and a waterfall that is not done when it looks done. */
class RequestRacesTest extends Playground {

    private static final String PAGE = "/app/request-races";

    @Test
    void theOlderAnswerWinsAndThePageLooksFinishedEitherWay() {
        open(PAGE);
        click("run-race");

        // The fast search answers first and both panels agree, which is the
        // moment a test that stops here would record as a pass.
        waitForFleetingText("naive-result", "fast");
        assertEquals("fast", text("guarded-result"));

        // Six hundred milliseconds later the older, slower answer lands and
        // overwrites the newer one in the unguarded panel.
        waitForText("naive-result", "slow");
        assertEquals("fast", text("guarded-result"), "the guarded panel answered a question nobody asked");
    }

    @Test
    void waitingForTheNetworkToGoQuietAgreesWithTheBug() {
        open(PAGE);
        waitForPresent("run-race");
        countRequestsInFlight();

        click("run-race");

        // Selenium has no equivalent of Playwright's networkidle, so this is a
        // hand-rolled one. It is worth writing out precisely because it is the
        // wait people reach for, and it is no help here -- it lands with no
        // spinner, no error and a plausible result, having answered a question
        // the user moved on from six hundred milliseconds ago.
        waitForNetworkQuiet();

        assertEquals("slow", text("naive-result"));
        assertEquals(
                "fast",
                text("guarded-result"),
                "asserting the result matches the last thing requested is what separates the two");
    }

    @Test
    void aWaterfallCostsTheSumOfItsStepsNotTheSlowest() {
        open(PAGE);
        click("run-waterfall");

        waitForPresent("waterfall-done");
        assertEquals(3, count("waterfall-step"));

        int total = Integer.parseInt(text("waterfall-total"));
        assertTrue(total > 700, "three sequential 250 ms steps cannot finish in " + total + " ms");
    }

    @Test
    void theStepsFinishInOrderBecauseEachWaitsForTheLast() {
        open(PAGE);
        click("run-waterfall");
        waitForPresent("waterfall-done");

        assertEquals(List.of("first", "second", "third"), textsOf("waterfall-step"));
    }

    @Test
    void waitingForTheFirstResponseReadsThePageTwoRequestsTooEarly() {
        open(PAGE);
        click("run-waterfall");

        // Playwright can wait on the response itself. WebDriver cannot see the
        // network without attaching CDP, so the DOM stands in for it: the first
        // rendered step is proof the first response landed, and it is exactly as
        // misleading a signal as the response event would have been.
        waitForPresent("waterfall-step");

        assertEquals(0, count("waterfall-done"), "the page cannot be finished with two requests still to go");
        assertNotEquals(3, count("waterfall-step"));
    }

    /** The trimmed text of every element carrying the test id, in document order. */
    private List<String> textsOf(String id) {
        return findAll(id).stream().map(WebElement::getText).map(String::trim).toList();
    }

    /**
     * Waits for a value that is only on screen for a moment.
     *
     * <p>{@code waitForText} polls every half second, which is fine for a value
     * that arrives and stays. Here "fast" holds the panel for the six hundred
     * milliseconds between the two answers, and a half-second poll can step
     * straight over it and report a failure that is really a sampling artefact.
     */
    private void waitForFleetingText(String id, String expected) {
        new WebDriverWait(driver, TIMEOUT, Duration.ofMillis(50)).until((ExpectedCondition<Boolean>) d -> {
            List<WebElement> found = d.findElements(testId(id));
            return !found.isEmpty() && found.get(0).getText().trim().equals(expected);
        });
    }

    /**
     * Starts counting requests the page has out.
     *
     * <p>Playwright is told about every request by the browser; WebDriver is not
     * told about any of them without attaching CDP, so the page has to be asked
     * to keep the tally itself. Counting completed entries out of the
     * Performance API instead is the tempting shortcut and it is wrong here:
     * this race has a six hundred millisecond gap in which nothing completes,
     * and a quiet-for-half-a-second rule declares victory inside it.
     */
    private void countRequestsInFlight() {
        ((JavascriptExecutor) driver).executeScript(
                "window.__inFlight = 0;"
                        + "const original = window.fetch;"
                        + "window.fetch = (...args) => {"
                        + "  window.__inFlight++;"
                        + "  return original(...args).finally(() => { window.__inFlight--; });"
                        + "};");
    }

    /** Blocks until the page has had nothing out for half a second. */
    private void waitForNetworkQuiet() {
        driver.manage().timeouts().scriptTimeout(TIMEOUT);
        ((JavascriptExecutor) driver).executeAsyncScript(
                "const done = arguments[arguments.length - 1];"
                        + "let quiet = 0;"
                        + "const tick = setInterval(() => {"
                        + "  if (window.__inFlight > 0) { quiet = 0; return; }"
                        + "  quiet += 100;"
                        + "  if (quiet >= 500) { clearInterval(tick); done(true); }"
                        + "}, 100);");
    }
}
