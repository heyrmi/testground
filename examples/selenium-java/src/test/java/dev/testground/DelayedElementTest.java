package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;

import org.junit.jupiter.api.Test;

/** /app/delayed-element — waiting for a condition rather than for a duration. */
class DelayedElementTest extends Playground {

    @Test
    void theMessageIsNotInTheDomUntilTheDelayElapses() {
        open("/app/delayed-element?delayMs=1500");

        assertEquals(0, count("delayed-message"));
        waitForPresent("delay-pending");
    }

    @Test
    void waitingForTheElementIsAllItTakes() {
        open("/app/delayed-element?delayMs=1500");

        // No sleep and no polling loop of our own: the wait retries the locate
        // until the element is there or the timeout expires.
        waitForText("delayed-message", "The element you were waiting for.");
    }

    @Test
    void theDelayIsUnderTheCallerControl() {
        open("/app/delayed-element?delayMs=0");

        waitForText("delay-ms", "0");
        waitForPresent("delayed-message");
    }

    @Test
    void restartingRemovesTheElementAndBringsItBack() {
        open("/app/delayed-element?delayMs=800");
        waitForPresent("delayed-message");

        click("restart");
        waitForAbsent("delayed-message");
        waitForPresent("delayed-message");
    }
}
