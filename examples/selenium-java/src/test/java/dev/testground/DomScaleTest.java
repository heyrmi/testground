package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.time.Duration;
import java.time.Instant;
import java.util.List;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.WebElement;

/** /app/dom-scale — a page whose only symptom is how long everything takes. */
class DomScaleTest extends Playground {

    private static final String PAGE = "/app/dom-scale";

    /** The cells are built outside React and carry no test id, so the manifest points at their host instead. */
    private static final By CELL = By.cssSelector(".scale-cell");

    @Test
    void thePageSaysNothingDifferentWhenItGetsHeavy() {
        open(PAGE + "?nodes=15000");
        waitForText("node-count", "0");

        click("build-nodes");
        waitForText("node-count", "15000");

        // Fifteen thousand nodes later every content assertion still passes and
        // not one of them mentions the cost. A suite that only reads content
        // would report this page as healthy right up to the timeout that kills
        // it.
        waitForText("thread-state", "free");
    }

    @Test
    void theCostIsInTheVolumeWhichIsExactNotInAStopwatch() {
        open(PAGE + "?nodes=25000");
        click("build-nodes");
        waitForText("node-count", "25000");

        // Twenty-five thousand elements a document-wide query has to walk, and
        // twenty-five thousand references it has to marshal back across the
        // wire. That is the cost, and it is a number rather than a feeling.
        assertEquals(25_000, driver.findElements(CELL).size());

        WebElement host = find("node-host");
        assertEquals(25_000, host.findElements(CELL).size());

        // Playwright reads every cell's text with one call into the page.
        // Selenium has no such call: asking each WebElement for its text would
        // be twenty-five thousand round trips and would take minutes rather
        // than milliseconds, so the honest equivalent is one script.
        @SuppressWarnings("unchecked")
        List<String> texts = (List<String>) ((JavascriptExecutor) driver).executeScript(
                "return Array.from(arguments[0].querySelectorAll('.scale-cell'), c => c.textContent);",
                host);
        assertEquals(25_000, texts.size());

        // Deliberately no wall-clock assertion. Timing one locator against
        // another inside a suite measures the machine it happens to be running
        // on, and a test that fails on a busy runner and passes on a laptop
        // teaches the wrong lesson. Time this by hand when you want the number.
        assertEquals(1, driver.findElements(By.cssSelector("[data-index='24999']")).size());
    }

    @Test
    void aBlockedThreadIsNotASlowResponseAndNeedsADifferentFix() {
        open(PAGE + "?blockMs=1500");

        Instant started = Instant.now();
        click("block-thread");

        // The page reports "blocked" while it is blocked, and no test will ever
        // see it: nothing in the page can run while the page is not running,
        // and every WebDriver command queues behind the same stalled renderer.
        // So the first thing readable after the click already says "free", and
        // the only evidence left is the clock.
        assertEquals("free", text("thread-state"));
        assertTrue(
                Duration.between(started, Instant.now()).toMillis() > 1200,
                "the thread was never actually blocked, so this challenge is not reproducing");
    }

    @Test
    void listenersAttachWithoutChangingAnythingThePageSays() {
        open(PAGE + "?nodes=2000");
        click("build-nodes");
        waitForText("node-count", "2000");

        click("attach-listeners");
        waitForText("listener-count", "500");
    }

    @Test
    void layoutThrashIsAStateNotAnAppearance() {
        open(PAGE + "?nodes=1500");
        click("build-nodes");
        waitForText("node-count", "1500");

        // A reflow per frame looks like nothing at all in a screenshot, so the
        // page publishes it as an attribute rather than leaving it to be seen.
        click("toggle-thrash");
        assertEquals("true", find("toggle-thrash").getAttribute("data-thrashing"));

        click("toggle-thrash");
        assertEquals("false", find("toggle-thrash").getAttribute("data-thrashing"));
    }

    @Test
    void theLeakIsOnlyObservableBecauseThisPageChoseToReportIt() {
        open(PAGE);

        for (int i = 0; i < 3; i++) {
            click("leak");
        }
        waitForText("retained-count", "3");

        // In a real application nothing would say this, which is the lesson:
        // the counter is the affordance, not the leak. Everything else on the
        // page is exactly as it was.
        waitForText("thread-state", "free");
    }
}
