package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.ElementClickInterceptedException;
import org.openqa.selenium.Keys;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.interactions.Actions;

/** /legacy/dialog-element — a modal dialog makes the background inert without hiding or disabling it. */
class DialogElementTest extends Playground {

    private static final String PAGE = "/legacy/dialog-element";

    @Test
    void aModalMakesTheBackgroundInertWhileLeavingItVisibleAndEnabled() {
        open(PAGE);
        click("open-modal");

        WebElement background = find("background-button");
        assertTrue(background.isDisplayed(), "the background is not hidden, only unreachable");
        assertTrue(background.isEnabled(), "nor is it disabled");

        // Every precondition a test would normally check has passed, and the
        // click is still refused. This reads as a flake and is the correct
        // behaviour: the backdrop is in the top layer and takes the hit.
        assertThrows(ElementClickInterceptedException.class, background::click);
        assertEquals("0", text("background-clicks"), "a click reached an inert background");
    }

    @Test
    void aNonModalDialogLeavesThePageWorking() {
        open(PAGE);
        click("open-modeless");

        // Same markup, same look, opened with show instead of showModal, and
        // the difference is entirely in whether the page behind it still works.
        assertTrue(find("modeless-dialog").isDisplayed());
        click("background-button");
        waitForText("background-clicks", "1");
    }

    @Test
    void theReturnValueIsTheOnlyRecordOfHowAModalClosed() {
        open(PAGE);

        click("open-modal");
        click("confirm-dialog");
        // The dialog's own contents are gone, so there is nothing left in it to
        // assert on. The return value it closed with is the whole record.
        assertFalse(find("modal-dialog").isDisplayed());
        waitForText("dialog-return", "confirmed");

        click("open-modal");
        click("cancel-dialog");
        waitForText("dialog-return", "cancelled");
    }

    @Test
    void escapeClosesAModalAndNothingElse() {
        open(PAGE);

        click("open-modal");
        pressEscape();
        assertFalse(find("modal-dialog").isDisplayed());
        // Escape is a real interaction here rather than a way to clear state,
        // and it leaves its own return value behind to prove it happened.
        waitForText("dialog-return", "escape");

        click("open-modeless");
        pressEscape();
        assertTrue(
                find("modeless-dialog").isDisplayed(),
                "escape closed a dialog nobody asked it to close");
    }

    /**
     * Sends Escape to whatever currently has focus.
     *
     * <p>Playwright addresses the keyboard directly. WebDriver has no page-level
     * keyboard, only elements and the low-level input queue, and an open modal
     * moves focus inside itself -- so sending the key to a fixed element such as
     * the body would deliver it somewhere the dialog is not listening. Actions
     * targets the active element, which is the one place both cases agree on.
     */
    private void pressEscape() {
        new Actions(driver).sendKeys(Keys.ESCAPE).perform();
    }
}
