package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.Keys;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.interactions.Actions;

/** /app/fake-controls — a switch, a rating and a slider with none of the elements a test reaches for. */
class FakeControlsTest extends Playground {

    private static final String PAGE = "/app/fake-controls";

    /**
     * The trap: {@code isSelected()} is the obvious way to ask a switch whether
     * it is on, and it answers about this one without ever looking at it.
     */
    @Test
    void theSwitchHasNoCheckboxToCheck() {
        open(PAGE);

        // Nothing to check, so nothing has a checkedness to report. WebDriver
        // answers false for a div rather than refusing the question, which is
        // why this assertion passes both before and after the switch is turned
        // on and would never fail no matter how broken the page became.
        assertEquals(0, driver.findElements(By.cssSelector("input[type=checkbox]")).size());
        assertFalse(find("toggle").isSelected());
        assertEquals("false", find("toggle").getAttribute("aria-checked"));

        click("toggle");

        assertFalse(find("toggle").isSelected(), "the switch grew a checkedness, so this trap is gone");
        // The state lives on the attributes, which is where it must be read.
        assertEquals("true", find("toggle").getAttribute("aria-checked"));
        assertEquals("on", find("toggle").getAttribute("data-state"));
        waitForText("toggle-state", "on");
    }

    @Test
    void theSwitchAnswersTheKeyboardToo() {
        open(PAGE);

        // The div carries a tabindex, so it is keyboard-interactable and takes
        // keys directly. Selenium has no focus() command of its own; sending
        // keys to the element is how focus gets there.
        find("toggle").sendKeys(Keys.SPACE);

        waitForText("toggle-state", "on");
    }

    @Test
    void readingTheStarsUnderThePointerMeasuresThePointer() {
        open(PAGE);

        new Actions(driver).moveToElement(find("star-4")).perform();
        waitForText("rating-shown", "4");

        // Nothing has been chosen. The stars are drawing the hover, and a test
        // that read them here would report a rating the page has never held.
        assertEquals("0", text("rating-value"));
    }

    @Test
    void movingThePointerAwayIsWhatMakesTheRatingReadable() {
        open(PAGE);

        click("star-3");
        // The row clears what it is drawing when the pointer leaves it, so
        // parking the pointer somewhere else is what makes the two agree.
        new Actions(driver).moveToElement(find("slider-value")).perform();

        waitForText("rating-value", "3");
        waitForText("rating-shown", "3");
    }

    @Test
    void theSliderHasNoValueToSetOnlyAPositionToDragTo() {
        open(PAGE);

        assertEquals(0, driver.findElements(By.cssSelector("input[type=range]")).size());
        waitForText("slider-value", "20");

        WebElement track = find("slider-track");
        int width = track.getSize().getWidth();

        // Moving to an element scrolls it into view; moving to bare screen
        // coordinates does not, so the same gesture written in absolute
        // positions would land on whatever happened to be there and report
        // nothing wrong.
        new Actions(driver)
                .moveToElement(track, -(width / 2) + (int) (width * 0.2), 0)
                .clickAndHold()
                .moveByOffset((int) (width * 0.55), 0)
                .release()
                .perform();

        int value = Integer.parseInt(text("slider-value"));
        assertTrue(value > 70 && value < 80, "the thumb ended at " + value + ", so the drag was not followed");
    }

    @Test
    void thereIsNoKeyboardShortcutForTheSliderEither() {
        open(PAGE);
        waitForText("slider-value", "20");

        // A native range answers the arrow keys. This one has no input behind
        // it and no key handler at all, so the keys go nowhere and -- as ever
        // with this challenge -- nothing reports a problem.
        ((JavascriptExecutor) driver).executeScript("arguments[0].focus();", find("slider-track"));
        new Actions(driver).sendKeys(Keys.ARROW_RIGHT, Keys.ARROW_RIGHT, Keys.END).perform();

        assertEquals("20", text("slider-value"), "the slider moved for the keyboard, so it has an input after all");
    }
}
