package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.WebElement;

/** /live/websocket — a socket that answers and a socket that just talks, and waiting for neither. */
class WebsocketTest extends Playground {

    private static final String PAGE = "/live/websocket";

    @Test
    void theEchoIsARoundTripSoTheClickAndTheReplyAreTwoEvents() {
        open(PAGE);

        click("echo-connect");
        waitForText("echo-state", "open");

        WebElement message = find("echo-input");
        message.clear();
        message.sendKeys("marco");
        click("echo-send");

        // The click sent a frame; it did not produce a reply. Reading echo-last
        // on the line after the click reads the page before the server has
        // spoken, and the value that is there -- "nothing yet" -- is a real
        // value rather than an error, so nothing complains.
        waitForText("echo-last", "echo: marco");
        waitForText("echo-count", "1");
    }

    @Test
    void sendingBeforeTheSocketIsOpenIsSilentlyDropped() {
        open(PAGE);

        // The trap that costs an afternoon: connect and send are two buttons,
        // and a test that presses them one after the other without waiting for
        // the state sends into a socket that is still connecting. The page
        // checks readyState, so the frame is dropped -- no error, no rejected
        // promise, nothing in the log. The failure surfaces later as a timeout
        // on the reply, pointing at the assertion rather than at the cause.
        click("echo-send");
        settle();
        assertEquals("nothing yet", text("echo-last"));
        assertEquals("0", text("echo-count"));

        // The same click, after the state says the socket is up, is answered.
        // That is what proves the silence above was the missing wait and not a
        // broken page.
        click("echo-connect");
        waitForText("echo-state", "open");
        click("echo-send");
        waitForText("echo-last", "echo: hello");
    }

    @Test
    void theTickerPushesWithNothingToWaitAfter() {
        // Nothing the test does causes the next message, so there is no action
        // to wait after -- only a condition to wait for. count is what turns a
        // moving number into a settled one: waiting for an exact value on a
        // counter that is still incrementing is a race the poll can lose in
        // both directions, arriving at four and then at six and never seeing
        // five at all.
        open(PAGE + "?ms=60&count=5");
        click("ticker-connect");

        waitForText("ticker-count", "5");
        waitForText("ticker-last-seq", "5");
    }

    @Test
    void theIntervalIsTheCallersToChooseSoASuiteNeedNotRunAtDemoSpeed() {
        // Half a second between pushes is right for a person watching and wrong
        // for a suite. The interval is a query parameter precisely so this test
        // costs a tenth of a second rather than two and a half.
        open(PAGE + "?ms=30&count=4");
        click("ticker-connect");

        waitForText("ticker-count", "4");
        waitForText("ticker-last-seq", "4");

        // count made the server stop, so the socket closes itself and the page
        // notices. Nobody clicked anything to cause this.
        waitForText("ticker-state", "closed");
    }

    @Test
    void theConnectionStateIsABetterSignalThanTheMessages() {
        open(PAGE + "?ms=40");
        click("ticker-connect");
        waitForText("ticker-state", "open");

        click("ticker-stop");
        waitForText("ticker-state", "closed");

        // Everything already received stays on screen after the socket is gone,
        // which is exactly why the messages cannot tell you whether it is still
        // there. A page that has stopped updating and a page that is between
        // updates render identically.
        String settled = text("ticker-count");
        settle();
        assertEquals(settled, text("ticker-count"), "a closed socket delivered another message");
        assertTrue(Integer.parseInt(settled) > 0, "the ticker never pushed anything to go stale");
    }

    @Test
    void theLogIsPresentAndEmptyUntilSomethingSpeaks() {
        open(PAGE + "?ms=40&count=3");

        // The list itself is always in the DOM; the lines are transient. That
        // distinction is the manifest's, and it matters: waiting for the log to
        // appear succeeds instantly and tells you nothing at all.
        waitForPresent("message-log");
        assertEquals(0, count("log-line"));

        click("ticker-connect");
        waitForText("ticker-count", "3");

        // Counting the lines rather than reading them keeps the assertion off
        // the generated names, which are the session's to choose.
        assertEquals(3, count("log-line"));
    }

    /**
     * Holds still for a moment.
     *
     * <p>The only sleep in this class, and it is here because the assertion
     * after it is that nothing happened. There is no condition that settles on
     * the absence of an event, so a wait would either return immediately and
     * prove nothing or need a duration anyway. Everywhere else in this suite a
     * wait retries until the page agrees.
     */
    private void settle() {
        try {
            Thread.sleep(400);
        } catch (InterruptedException interrupted) {
            Thread.currentThread().interrupt();
            throw new IllegalStateException("interrupted while holding still", interrupted);
        }
    }
}
