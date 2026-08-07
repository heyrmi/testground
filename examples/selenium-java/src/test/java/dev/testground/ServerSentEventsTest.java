package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.time.Duration;
import java.util.List;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.TimeoutException;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.support.ui.WebDriverWait;

/** /live/server-sent-events — a stream that ends, one that stalls forever, and one that writes. */
class ServerSentEventsTest extends Playground {

    private static final String PAGE = "/live/server-sent-events";

    /** The text the token stream spells out. Fixed on the server so it can be asserted whole. */
    private static final String COMPLETE_TEXT =
            "A stream is not a page. It arrives in pieces, each one a complete and correct view "
                    + "of nothing in particular, and the only moment the whole thing is true is "
                    + "after the last piece lands.";

    @Test
    void aStreamThatFinishesSaysSo() {
        open(PAGE + "?count=4&ms=40");
        click("events-start");

        // The terminating event is the signal, and the page publishes it as a
        // state rather than leaving a reader to infer an ending from silence.
        // Assert on that first: it is the only one of the two that distinguishes
        // this stream from the stalled one below.
        waitForText("events-state", "done");
        waitForText("events-count", "4");
    }

    @Test
    void aStalledStreamIsNeitherFailedNorFinished() {
        open(PAGE + "?before=3&ms=40");
        click("stall-start");

        waitForText("stall-count", "3");

        // Three updates and then nothing, with the connection still open. No
        // error was raised, no close arrived, no done event is coming, and the
        // page is showing a partial result that is indistinguishable on screen
        // from a complete one. Everything about it looks like success.
        settle();
        assertEquals("3", text("stall-count"), "the stalled stream sent a fourth update");
        assertEquals("streaming", text("stall-state"));
        assertNotEquals("done", text("stall-state"));
    }

    @Test
    void waitingForTheUpdateThatIsNotComingBlamesTheUpdate() {
        open(PAGE + "?before=2&ms=30");
        click("stall-start");
        waitForText("stall-count", "2");

        // The trap, and it is the expensive kind because it looks like the
        // right test. Waiting for the third update spends the whole timeout and
        // then fails naming the counter -- "stall-count was never 3" -- which
        // sends the next reader to look for a rendering bug. The stream is what
        // is wrong, and nothing in this failure says so.
        //
        // A short wait of its own, because the point is the timeout and paying
        // the suite's full ten seconds to make it would be its own bad habit.
        WebDriverWait impatient = new WebDriverWait(driver, Duration.ofSeconds(2));
        assertThrows(
                TimeoutException.class,
                () -> impatient.until(d -> "3".equals(d.findElement(testId("stall-count")).getText().trim())));

        // Assert what you actually expect instead. "It stops at two and stays
        // open" is a claim about the stream, it passes in two seconds rather
        // than failing in ten, and when it breaks it breaks for a reason.
        assertEquals("2", text("stall-count"));
        assertEquals("streaming", text("stall-state"));
    }

    @Test
    void everyIntermediateStateOfTheTokenStreamReadsCorrectly() {
        open(PAGE + "?ms=60");
        click("stream-start");

        // Grabbed while the stream is still running, so this is a genuine
        // intermediate render rather than the finished one read twice.
        waitForAtLeast("stream-tokens", 1);
        String partial = text("stream-text");
        assertEquals("streaming", text("stream-state"));

        waitForText("stream-state", "done");
        String complete = text("stream-text");

        // The partial was a real sentence, correctly spelled and correctly
        // punctuated. It was simply not the whole one, and nothing about
        // reading it could have told you that.
        assertTrue(partial.length() > 0, "nothing had arrived, so there was no partial state to catch");
        assertTrue(
                complete.length() > partial.length(),
                "the stream was already finished when the partial was read, so this proves nothing");
        assertEquals(COMPLETE_TEXT, complete);
    }

    @Test
    void waitingForTheDoneStateNotForASubstringThatWasTrueEarlier() {
        open(PAGE + "?ms=60");
        click("stream-start");

        // The tempting assertion, and it passes -- about a fifth of the way in.
        // A test that stops here has waited for a sentence the stream said long
        // before it had finished saying anything, and it will happily pass
        // against a stream that dies immediately afterwards.
        waitForTextContaining("stream-text", "A stream is not a page");
        assertEquals("streaming", text("stream-state"), "the substring only became true at the end");

        // The state is the assertion that means "finished". Only after it does
        // the tail exist, and the tail is the part a stalled stream never
        // reaches.
        waitForText("stream-state", "done");
        assertTrue(text("stream-text").endsWith("after the last piece lands."));
    }

    /** Waits until an element's text contains the fragment, and not merely until it exists. */
    private void waitForTextContaining(String id, String fragment) {
        wait.until(d -> {
            List<WebElement> found = d.findElements(testId(id));
            return !found.isEmpty() && found.get(0).getText().contains(fragment);
        });
    }

    /** Waits until a numeric cell has climbed to at least {@code minimum}. */
    private int waitForAtLeast(String id, int minimum) {
        return wait.until(d -> {
            List<WebElement> found = d.findElements(testId(id));
            if (found.isEmpty()) {
                return null;
            }
            int value = Integer.parseInt(found.get(0).getText().trim());
            return value >= minimum ? value : null;
        });
    }

    /**
     * Holds still for a moment.
     *
     * <p>The only sleep in this class, and the challenge is the reason for it:
     * the claim being made about the stalled stream is that nothing further
     * arrives, and no condition ever settles on an event not happening.
     */
    private void settle() {
        try {
            Thread.sleep(600);
        } catch (InterruptedException interrupted) {
            Thread.currentThread().interrupt();
            throw new IllegalStateException("interrupted while holding still", interrupted);
        }
    }
}
