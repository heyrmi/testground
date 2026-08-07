package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;

import java.time.Duration;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.interactions.Actions;

/** /app/pointer-menus — menus that only exist while the pointer is on them, and gestures that are not clicks. */
class PointerMenusTest extends Playground {

    private static final String PAGE = "/app/pointer-menus";

    @Test
    void theMenuExistsOnlyWhileThePointerIsInsideTheGroup() {
        open(PAGE);
        assertEquals(0, count("hover-menu"), "the menu should not be in the DOM before anything is hovered");

        hover("hover-trigger");
        waitForPresent("hover-menu");

        // Moving anywhere outside the wrapper removes it, so a test that hovers,
        // then goes looking somewhere else, then comes back to click has already
        // lost the menu.
        hover("gesture-target");
        waitForAbsent("hover-menu");
    }

    @Test
    void anItemIsChosenByMovingOntoTheMenuRatherThanAwayFromIt() {
        open(PAGE);

        hover("hover-trigger");
        // The pointer travels from the trigger into the menu without leaving the
        // wrapper, which is the only route that keeps the menu alive long enough
        // to be clicked.
        new Actions(driver).moveToElement(find("menu-open")).click().perform();

        waitForText("menu-choice", "open");
    }

    @Test
    void theSubmenuNeedsTheParentHoveredFirst() {
        open(PAGE);

        hover("hover-trigger");
        assertEquals(0, count("submenu"), "the submenu should wait for its own row to be hovered");

        hover("menu-more");
        waitForPresent("submenu");

        new Actions(driver).moveToElement(find("menu-archive")).click().perform();
        waitForText("menu-choice", "archive");
    }

    @Test
    void rightClickRaisesThePageOwnMenuNotTheBrowserOne() {
        open(PAGE);

        // If the page did not suppress the native menu this would be the end of
        // the run: Chrome's own context menu is drawn outside the page and
        // WebDriver has nothing that can dismiss it.
        new Actions(driver).contextClick(find("gesture-target")).perform();
        waitForPresent("context-menu");

        click("context-rename");
        waitForText("menu-choice", "rename");
        waitForAbsent("context-menu");
    }

    @Test
    void countingADoubleClickAsOneClickIsTheAssertionThatLooksRight() {
        open(PAGE);

        new Actions(driver).doubleClick(find("gesture-target")).perform();

        waitForText("double-clicks", "1");

        // The tempting assertion here is that one click happened. It reads well
        // and it is false: a double click delivers two complete click events on
        // the way, and a page that acts on both will act twice.
        waitForText("single-clicks", "2");
    }

    @Test
    void aHoldIsThreeStepsBecauseAClickIsAllOfThemAtOnce() {
        open(PAGE);
        WebElement target = find("gesture-target");

        new Actions(driver).click(target).perform();
        assertEquals("0", text("long-presses"), "an ordinary click is far too short to be a hold");

        // The pause belongs inside the chain rather than as a Java-side sleep
        // between two perform() calls: one chain is one W3C action sequence, so
        // the button is genuinely held for the whole 650 ms instead of for
        // however long two driver round trips happened to take.
        new Actions(driver)
                .moveToElement(target)
                .clickAndHold()
                .pause(Duration.ofMillis(650))
                .release()
                .perform();

        waitForText("long-presses", "1");
    }

    /**
     * Puts the pointer on an element.
     *
     * <p>Playwright's hover() waits for the element on its own; {@code Actions}
     * takes an element that must already exist, so the wait is find()'s job and
     * the move is a separate step.
     */
    private void hover(String id) {
        new Actions(driver).moveToElement(find(id)).perform();
    }
}
