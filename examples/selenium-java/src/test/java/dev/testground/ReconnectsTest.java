package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.List;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.WebElement;

/** /live/reconnects — a socket that dies quietly, and messages that arrive out of their order. */
class ReconnectsTest extends Playground {

    private static final String PAGE = "/live/reconnects";

    @Test
    void aDroppedSocketIsAnnouncedByTheStateNotByTheContent() {
        open(PAGE + "?dropAfterMs=800");
        click("flaky-connect");

        int before = waitForAtLeast("flaky-count", 1);
        waitForAtLeast("flaky-drops", 1);

        // The connection is gone and the messages are still on the screen,
        // unchanged and perfectly plausible. A test that only reads content is
        // looking at a page that has merely stopped moving, which is what a
        // healthy page looks like between updates. Nothing here has failed
        // loudly enough for anything to notice.
        assertTrue(Integer.parseInt(text("flaky-count")) >= before);

        // The generation is the part that says something happened: the client
        // came back, so what is on screen after this point came from a
        // different connection than what was on screen before it.
        waitForAtLeast("flaky-generation", 2);
    }

    @Test
    void reconnectingRecoversAndTheGenerationSaysWhichConnectionYouAreReading() {
        open(PAGE + "?dropAfterMs=800");
        click("flaky-connect");

        // "At least", not "exactly". Both of these are counters that are still
        // moving, and pinning an exact value on one of those is the same race
        // the ticker teaches next door -- a poll can arrive at one and again at
        // three and never see two. The lesson is that the drops happened and
        // the client came back, not that it happened precisely twice.
        waitForAtLeast("flaky-drops", 2);
        int generation = waitForAtLeast("flaky-generation", 3);

        // The invariant worth holding on to: every drop the client wanted back
        // from produced one more connection than there were drops.
        assertTrue(generation >= 3, "the client stopped reconnecting after the second drop");

        // Messages accumulate across connections, so the total alone can never
        // tell you the socket died -- it only ever goes up, either way.
        assertTrue(Integer.parseInt(text("flaky-count")) > 0);
    }

    @Test
    void stoppingDeliberatelyIsDistinguishableFromBeingDropped() {
        open(PAGE + "?dropAfterMs=5000");
        click("flaky-connect");
        waitForText("flaky-state", "open");

        // Both endings close the socket and both bump the drop count. Only the
        // state separates "I asked for this" from "it fell over": a deliberate
        // stop lands on closed, a drop the client wants back from lands on
        // reconnecting.
        click("flaky-stop");
        waitForText("flaky-state", "closed");

        // And it stays stopped. Without this the assertion above is equally
        // true of a client that is merely between attempts, which is the state
        // it would be in for a hundred milliseconds after any ordinary drop.
        String generation = text("flaky-generation");
        settle();
        assertEquals("closed", text("flaky-state"));
        assertEquals(generation, text("flaky-generation"), "the client reconnected after being told not to");
    }

    @Test
    void messagesArriveInAnOrderThatIsNotTheirNumbering() {
        open(PAGE + "?count=6");

        // The container is in the DOM from the start and empty; the done marker
        // is added inside it. Waiting for the container therefore succeeds
        // before the socket has even opened and proves nothing whatsoever.
        assertEquals(1, count("shuffled-outcome"));
        assertEquals(0, count("shuffled-done"));

        click("shuffled-connect");
        waitForPresent("shuffled-done");

        // The reordering is fixed rather than random, so this is an exact
        // assertion rather than a hopeful one.
        assertEquals("2, 1, 4, 3, 6, 5", text("arrival-order"));
        assertEquals("1, 2, 3, 4, 5, 6", text("sorted-order"));
    }

    @Test
    void appendingInArrivalOrderRendersThemWrong() {
        open(PAGE + "?count=4");
        click("shuffled-connect");
        waitForPresent("shuffled-done");

        String arrival = text("arrival-order");
        String sorted = text("sorted-order");

        // Both lists hold the same four messages and only one of them is what a
        // reader should see. Sorting by the sequence the server stamped, rather
        // than trusting the order the frames turned up in, is the entire fix --
        // and a page that renders the arrival order looks completely fine until
        // somebody reads it.
        assertEquals("1, 2, 3, 4", sorted);
        assertNotEquals(sorted, arrival, "the socket delivered in order, so this proves nothing");
    }

    /**
     * Waits until a numeric cell has reached at least {@code minimum}, and
     * returns what it read.
     *
     * <p>The shape every assertion on this page wants. These counters only ever
     * climb, so a lower bound settles where an equality would be a race against
     * the next increment.
     */
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
     * <p>The only sleep in this class. The assertion after it is that nothing
     * happened -- no reconnect, no new generation -- and there is no condition
     * that settles on an event failing to occur.
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
