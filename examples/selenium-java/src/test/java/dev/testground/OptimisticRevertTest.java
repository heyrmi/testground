package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.List;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.support.ui.ExpectedCondition;

/** /app/optimistic-revert — settling a write before believing what the row says. */
class OptimisticRevertTest extends Playground {

    /**
     * A latency long enough that the in-flight window survives a WebDriver round
     * trip. The Playwright suite gets away with 300 ms because its assertions run
     * in-process; every command here is an HTTP call to the driver, so a window
     * that short can close before we ever look at it. Slowing the server down is
     * the honest fix — shortening the wait would only hide the race.
     */
    private static final String SLOW = "?latencyMs=1200";

    @Test
    void theEndpointSaysInAdvanceWhichTasksTheServerWillRefuse() {
        // Knowing this up front is what lets the tests below commit to an
        // expected outcome instead of asserting whichever one they observed.
        open("/api/app/optimistic-revert/tasks");
        String body = driver.getPageSource();

        assertTrue(body.contains("\"id\":3,") && body.contains("\"id\":6,"), "the seeded task ids changed");
        assertEquals(2, countOccurrences(body, "\"rejects\":true"), "exactly every third task should be locked");
    }

    @Test
    void aWriteTheServerAcceptsSticks() {
        open("/app/optimistic-revert" + SLOW);

        clickInRow(1, "task-toggle");
        waitForRowPresent(1, "task-saving");
        waitForRowAbsent(1, "task-saving");

        waitForRowText(1, "task-state", "done");
        assertEquals(0, count("revert-notice"), "an accepted write should not have raised a revert notice");
    }

    @Test
    void aWriteTheServerRefusesFlipsBackOnItsOwn() {
        open("/app/optimistic-revert" + SLOW);

        clickInRow(3, "task-toggle");

        // The optimistic value is real and on screen; it is simply not agreed
        // yet. Row 3 is one the tasks endpoint publishes as locked.
        waitForRowText(3, "task-state", "done");

        // Waiting for the write to settle is what makes the next line mean
        // anything. The saving marker is present for exactly the round trip.
        waitForRowAbsent(3, "task-saving");
        waitForRowText(3, "task-state", "todo");
        waitForPresent("revert-notice");
        waitForText("rejected-count", "1");
    }

    @Test
    void assertingOnTheClickAloneGivesAGreenRunForABrokenWrite() {
        open("/app/optimistic-revert?latencyMs=2500");

        clickInRow(3, "task-toggle");

        // This assertion passes and the write did not happen. It is the trap the
        // page exists to teach: a value the client invented reads exactly like a
        // value the server agreed to, and nothing in a run that stops here says
        // which one it saw.
        waitForRowText(3, "task-state", "done");

        // Left in to prove the first assertion was lying rather than merely
        // early.
        waitForRowText(3, "task-state", "todo");
    }

    @Test
    void theServerIsTheSourceOfTruthWhateverTheDomShowed() {
        open("/app/optimistic-revert?latencyMs=0");

        clickInRow(3, "task-toggle");
        waitForText("rejected-count", "1");

        // Selenium has no request-scoped API client the way Playwright's `page`
        // does, so the check is a navigation to the same endpoint. The pinned
        // session cookie rides along, which is what makes it this test's tasks
        // and not somebody else's.
        open("/api/app/optimistic-revert/tasks");
        assertTrue(
                driver.getPageSource().contains("\"id\":3,\"title\":"),
                "task 3 was missing from the authoritative list");
        assertTrue(
                !driver.getPageSource().contains("\"done\":true"),
                "the server recorded a toggle it told the page it had refused");
    }

    /**
     * A test id inside one task row.
     *
     * <p>The row is narrowed by {@code data-task-id}, which the challenge
     * publishes for exactly this purpose. Playwright scopes with a locator
     * object; Selenium has no equivalent that survives a re-render, so the scope
     * is folded into the selector and re-resolved on every poll.
     */
    private static By inRow(int taskId, String id) {
        return By.cssSelector("[data-testid='task'][data-task-id='" + taskId + "'] [data-testid='" + id + "']");
    }

    private void clickInRow(int taskId, String id) {
        wait.until(d -> {
            List<WebElement> found = d.findElements(inRow(taskId, id));
            return found.isEmpty() ? null : found.get(0);
        }).click();
    }

    private void waitForRowText(int taskId, String id, String expected) {
        wait.until((ExpectedCondition<Boolean>) d -> {
            List<WebElement> found = d.findElements(inRow(taskId, id));
            return !found.isEmpty() && found.get(0).getText().trim().equals(expected);
        });
    }

    private void waitForRowPresent(int taskId, String id) {
        wait.until((ExpectedCondition<Boolean>) d -> !d.findElements(inRow(taskId, id)).isEmpty());
    }

    private void waitForRowAbsent(int taskId, String id) {
        wait.until((ExpectedCondition<Boolean>) d -> d.findElements(inRow(taskId, id)).isEmpty());
    }

    private static int countOccurrences(String haystack, String needle) {
        int found = 0;
        for (int at = haystack.indexOf(needle); at >= 0; at = haystack.indexOf(needle, at + needle.length())) {
            found++;
        }
        return found;
    }
}
