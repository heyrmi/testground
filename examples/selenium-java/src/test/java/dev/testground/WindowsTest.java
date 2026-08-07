package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import java.util.HashSet;
import java.util.Set;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.NoSuchWindowException;

/** /legacy/windows — a new tab is a context of its own, and every locator keeps pointing at the old one. */
class WindowsTest extends Playground {

    private static final String PAGE = "/legacy/windows";

    @Test
    void aNewTabIsAContextYourLocatorsAreNotPointingAt() {
        open(PAGE);
        String opener = driver.getWindowHandle();
        String tab = windowOpenedBy("blank-link");

        // The tab exists and is fully loaded, and every locator in this test is
        // still addressing the opener, where none of its content exists. This is
        // the quiet failure the challenge is about: the popup works perfectly
        // and the assertion runs against the wrong document.
        assertEquals(0, count("popup-target"));

        driver.switchTo().window(tab);
        assertEquals("Popup: tab", text("popup-target"));

        driver.close();

        // And a test that finishes pointing at a window it closed fails on its
        // next step, for reasons that have nothing to do with the next step.
        // find() would hide this: NoSuchWindowException is a NotFoundException,
        // so the base class's wait swallows it and reports a timeout instead.
        assertThrows(NoSuchWindowException.class, () -> driver.findElement(testId("from-popup")));

        driver.switchTo().window(opener);
        assertEquals("nothing yet", text("from-popup"), "the opener was never touched");
    }

    @Test
    void windowOpenWithDimensionsOpensTheSameWay() {
        open(PAGE);
        String opener = driver.getWindowHandle();
        String popup = windowOpenedBy("open-popup");

        // A sized window.open and a target=_blank link produce the same thing
        // as far as WebDriver is concerned: one more handle in the set.
        driver.switchTo().window(popup);
        assertEquals("Popup: sized", text("popup-target"));

        driver.close();
        driver.switchTo().window(opener);
    }

    @Test
    void aPopupThatClosesItselfHasToBeReadQuickly() {
        open(PAGE);
        String opener = driver.getWindowHandle();
        String popup = windowOpenedBy("open-closing");

        driver.switchTo().window(popup);
        assertEquals("Popup: closing", text("popup-target"));

        // Switch back before it goes, so the wait below is running from a window
        // that will still be there when the other one is not.
        driver.switchTo().window(opener);
        wait.until(d -> !d.getWindowHandles().contains(popup));

        // Too late now, and the error is about a closed target rather than about
        // timing -- which is what makes this read as a different bug entirely.
        assertThrows(NoSuchWindowException.class, () -> driver.switchTo().window(popup));
    }

    @Test
    void theOpenerOutlivesThePopupWhichMakesItTheBetterTarget() {
        open(PAGE);
        click("open-writer");

        // No handle, no switching, no race with the popup's own closing. The
        // popup reached into this page and then went; asserting here needs none
        // of the machinery the other three tests spend their time on.
        waitForText("from-popup", "written by the popup");
    }

    /**
     * Clicks something that opens a window and returns the new handle.
     *
     * <p>Playwright has to subscribe to the page event before the click, because
     * afterwards the event has already been delivered. WebDriver has no such
     * event at all, so the handle set has to be captured before the click for a
     * different reason and with the same discipline: afterwards there is nothing
     * left to diff against.
     */
    private String windowOpenedBy(String id) {
        Set<String> before = new HashSet<>(driver.getWindowHandles());
        click(id);
        return wait.until(d -> {
            Set<String> opened = new HashSet<>(d.getWindowHandles());
            opened.removeAll(before);
            return opened.isEmpty() ? null : opened.iterator().next();
        });
    }
}
