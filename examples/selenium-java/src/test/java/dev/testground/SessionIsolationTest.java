package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.Cookie;

/**
 * Not a challenge: the property every other class in this suite depends on.
 *
 * <p>If sessions were not isolated, two of these tests toggling the same task
 * would fight, and the suite would be flaky for a reason that had nothing to do
 * with the challenges. Worth proving once, explicitly, rather than inferring it
 * from the absence of trouble.
 */
class SessionIsolationTest extends Playground {

    @Test
    void thePinnedSessionIsTheOneTheServerUses() {
        open("/api/control/state");

        Cookie cookie = driver.manage().getCookieNamed("playground_session");
        assertEquals("se-SessionIsolationTest", cookie.getValue());
        assertTrue(
                driver.getPageSource().contains("se-SessionIsolationTest"),
                "the server reported a different session than the cookie pinned");
    }

    @Test
    void twoSessionsDoNotSeeEachOtherMutations() {
        // Flip the first row in this test's own session.
        //
        // Waiting for the row to read "done" is NOT enough, and getting that
        // wrong here is the challenge working as designed: the page applies the
        // toggle optimistically, so the row says done while the request is
        // still in flight, and navigating away at that moment cancels the very
        // request the next assertion depends on. The saving marker is the
        // honest signal -- it is present for exactly as long as the round trip.
        open("/app/optimistic-revert?latencyMs=200");
        waitForPresent("task-toggle");
        assertEquals("todo", text("task-state"));
        click("task-toggle");
        waitForPresent("task-saving");
        waitForAbsent("task-saving");
        assertEquals("done", text("task-state"));

        open("/api/app/optimistic-revert/tasks");
        String mine = driver.getPageSource();

        // Move to a session nobody else uses and read the same endpoint. The
        // toggle above must be invisible from here: same seed, same tasks, and
        // none of them done.
        driver.manage().addCookie(new Cookie("playground_session", "se-isolation-neighbour", "/"));
        driver.navigate().refresh();
        String theirs = driver.getPageSource();

        assertNotEquals(
                mine,
                theirs,
                "a second session saw the first session's toggle, so the workers are not isolated");
        assertTrue(
                theirs.contains("\"done\":false") && !theirs.contains("\"done\":true"),
                "the neighbouring session had a completed task, which only this test completed");
    }
}
