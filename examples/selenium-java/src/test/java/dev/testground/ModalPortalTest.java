package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.ElementClickInterceptedException;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.Keys;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.interactions.Actions;

/** /app/modal-portal — a dialog rendered onto the body, and everything that follows from it. */
class ModalPortalTest extends Playground {

    private static final String PAGE = "/app/modal-portal";

    @Test
    void theDialogIsNotInsideTheApplicationRoot() {
        open(PAGE);
        click("open-modal");

        assertTrue(find("modal").isDisplayed());

        // Scoping to the component tree finds nothing, which is the whole trap:
        // the dialog is plainly on screen and a page-object rooted at the
        // application's own container will insist it does not exist.
        assertEquals(0, driver.findElements(By.cssSelector("#root [data-testid='modal']")).size());
        assertEquals(1, driver.findElements(By.cssSelector("body > [data-testid='modal-overlay']")).size());
    }

    /**
     * The trap that reads as a broken locator: the background button is present,
     * enabled and visible, and clicking it fails naming something the test never
     * mentioned.
     */
    @Test
    void theBackgroundIsEnabledVisibleAndUnclickable() {
        open(PAGE);
        click("open-modal");

        WebElement background = find("background-button");
        assertTrue(background.isEnabled());
        assertTrue(background.isDisplayed());

        ElementClickInterceptedException refused =
                assertThrows(ElementClickInterceptedException.class, background::click);

        // Treat it as information rather than as flake: the overlay really is on
        // top, and a user would have exactly the same problem.
        assertTrue(
                refused.getMessage().contains("modal-overlay"),
                "the error should name what is in the way, but said: " + refused.getMessage());
        assertEquals("0", sourceText("background-clicks"));
    }

    @Test
    void thePageBehindCannotScrollAndSaysSo() {
        open(PAGE);
        waitForText("scroll-state", "free");

        click("open-modal");
        waitForSourceText("scroll-state", "locked");

        // Published on the body, so the lock can be asserted rather than
        // inferred from a scroll that silently did nothing.
        WebElement body = driver.findElement(By.tagName("body"));
        assertEquals("true", body.getAttribute("data-scroll-locked"));

        // A wheel gesture is what a user has, and the lock takes it away.
        // Scripted scrolling would not show this: overflow:hidden stops the
        // user and leaves window.scrollTo working, so a test that reached for
        // JavaScript here would conclude the lock was not there at all.
        long locked = scrollY();
        new Actions(driver).scrollByAmount(0, 400).perform();
        assertEquals(locked, scrollY(), "the page behind scrolled, so the lock is not holding");

        click("modal-cancel");
        waitForText("scroll-state", "free");
        assertNull(driver.findElement(By.tagName("body")).getAttribute("data-scroll-locked"));

        // The same gesture once the lock has gone, so the no-op above is
        // attributable to the lock rather than to the gesture never working.
        new Actions(driver).scrollByAmount(0, 400).perform();
        assertTrue(scrollY() > locked, "the wheel gesture does nothing even unlocked, so it proved nothing above");
    }

    @Test
    void tabCannotLeaveTheDialog() {
        open(PAGE);
        click("open-modal");
        assertEquals("modal-confirm", focusedTestId());

        // Selenium sends keys to whatever holds focus, so the trap is exercised
        // the same way a person would exercise it.
        new Actions(driver).sendKeys(Keys.TAB).perform();
        assertEquals("modal-cancel", focusedTestId());

        // A fixed number of tabs walks out of any normal page and lands
        // somewhere the test never predicted. Not this one: it comes back round.
        new Actions(driver).sendKeys(Keys.TAB).perform();
        assertEquals("modal-confirm", focusedTestId());
    }

    @Test
    void theDialogReportsHowItClosedBecauseAfterwardsThereIsNothingToAsk() {
        open(PAGE);

        click("open-modal");
        click("modal-confirm");
        waitForText("modal-outcome", "confirmed");

        click("open-modal");
        click("modal-cancel");
        waitForText("modal-outcome", "cancelled");

        click("open-modal");
        new Actions(driver).sendKeys(Keys.ESCAPE).perform();
        waitForText("modal-outcome", "escape");
    }

    @Test
    void clickingTheOverlayItselfClosesTheDialog() {
        open(PAGE);
        click("open-modal");

        // The overlay covers the viewport, so its centre is the dialog: a plain
        // click() lands on the dialog, which is not the overlay itself and
        // therefore closes nothing. The corner is the only part of it that is
        // actually the overlay.
        WebElement overlay = find("modal-overlay");
        new Actions(driver)
                .moveToElement(
                        overlay,
                        -(overlay.getSize().getWidth() / 2) + 8,
                        -(overlay.getSize().getHeight() / 2) + 8)
                .click()
                .perform();

        waitForText("modal-outcome", "overlay");
    }

    /** Which of the published test ids currently holds focus. */
    private String focusedTestId() {
        return driver.switchTo().activeElement().getAttribute("data-testid");
    }

    /**
     * An element's source text, which is the only kind readable while the
     * dialog is open.
     *
     * <p>The scroll lock puts {@code overflow: hidden} on the body, and
     * WebDriver then treats everything below the fold as clipped away:
     * {@code isDisplayed()} answers false and {@code getText()} answers with an
     * empty string for a readout that is still plainly saying "locked". So the
     * lock costs a second thing beyond the scrolling — the page's own status
     * stops being readable at exactly the moment a test wants to read it.
     * Playwright's text assertions go through textContent and never meet this.
     */
    private String sourceText(String id) {
        return find(id).getDomProperty("textContent").trim();
    }

    private void waitForSourceText(String id, String expected) {
        wait.until(d -> {
            var found = d.findElements(testId(id));
            return !found.isEmpty() && expected.equals(found.get(0).getDomProperty("textContent").trim());
        });
    }

    private long scrollY() {
        return (Long) ((JavascriptExecutor) driver).executeScript("return Math.round(window.scrollY);");
    }
}
