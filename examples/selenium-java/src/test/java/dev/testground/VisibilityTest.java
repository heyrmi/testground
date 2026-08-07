package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.ElementClickInterceptedException;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.support.ui.ExpectedCondition;

/** /legacy/visibility — seven buttons that are all present, and seven different answers to "is it there?". */
class VisibilityTest extends Playground {

    private static final String PAGE = "/legacy/visibility";

    @Test
    void theControlIsPlainlyClickable() {
        open(PAGE);

        click("btn-normal");
        waitForText("clicked", "normal");
    }

    @Test
    void anInvisibleButtonCanStillBePerfectlyClickable() {
        open(PAGE);

        // Where the two frameworks disagree, and the disagreement is the point.
        // Playwright calls opacity:0 visible; Selenium folds opacity into
        // isDisplayed and calls it hidden. Neither answer stops the click,
        // because the browser delivers one to a transparent element quite
        // happily -- so a suite that trusts either check has still just done
        // something no user could have done.
        assertFalse(find("btn-opacity-zero").isDisplayed());

        click("btn-opacity-zero");
        waitForText("clicked", "opacity-zero");
    }

    @Test
    void displayNoneAndVisibilityHiddenAreBothHiddenAndDifferInLayout() {
        open(PAGE);

        assertFalse(find("btn-display-none").isDisplayed());
        assertFalse(find("btn-visibility-hidden").isDisplayed());

        // The same answer from isDisplayed, and a completely different effect on
        // the page: one is out of the layout, the other still occupies its box.
        // Read from the DOM rather than from getRect, which reports a full-size
        // box for the display:none button as well and so cannot tell them apart
        // -- exactly the check a test would reach for first.
        assertEquals(0.0, boxOf("btn-display-none", "width"));
        assertTrue(boxOf("btn-visibility-hidden", "width") > 0);
    }

    @Test
    void aZeroSizeButtonIsHiddenByHavingNowhereToBeClicked() {
        open(PAGE);

        // Nothing about its styling says hidden. It is hidden because there is
        // no pixel belonging to it for a click to land on.
        assertFalse(find("btn-zero-size").isDisplayed());
        assertEquals(0.0, boxOf("btn-zero-size", "width"));
        assertEquals(0.0, boxOf("btn-zero-size", "height"));
    }

    @Test
    void anOffScreenButtonIsLaidOutSizedAndUnreachableByAPerson() {
        open(PAGE);

        // The other way round from opacity:0, and just as instructive: here
        // Playwright calls the element visible and Selenium calls it hidden.
        // Arguing about the verdict misses the point -- the measurable facts
        // agree, and they are the ones worth asserting on.
        assertFalse(find("btn-offscreen").isDisplayed());
        assertTrue(boxOf("btn-offscreen", "width") > 0, "it has a real size, which is why size checks pass");
        assertTrue(boxOf("btn-offscreen", "left") < 0, "and it is nowhere near the viewport");
    }

    @Test
    void aCoveredButtonPassesEveryVisibilityCheckAndTheClickHitsTheOverlay() {
        open(PAGE);
        WebElement covered = find("btn-covered");

        assertTrue(covered.isDisplayed());
        assertTrue(covered.isEnabled());

        // The failure names an element the test never mentioned. Believe it:
        // this is a real bug, and users are hitting the same overlay.
        ElementClickInterceptedException intercepted =
                assertThrows(ElementClickInterceptedException.class, covered::click);
        assertTrue(
                intercepted.getMessage().contains("overlay"),
                "the interception did not say what took the click: " + intercepted.getMessage());
        assertEquals("none", text("clicked"));
    }

    @Test
    void aTransitioningButtonIsInPlaceBeforeItIsInPlace() {
        open(PAGE);
        click("reveal");

        // Mid-transition it is neither hidden nor settled: partly transparent,
        // still sliding, and the coordinates any click would use are out of date
        // by the time the click is dispatched.
        assertTrue(opacityOf("btn-fading") < 1.0);

        // Playwright waits for an element to stop moving before it clicks.
        // WebDriver has no such check -- it scrolls, computes one point and
        // clicks it -- so waiting the element out is the test's job here. This
        // is the clearest case in this batch of an actionability guarantee one
        // tool gives away and the other leaves to the caller.
        waitUntilStill("btn-fading");
        click("btn-fading");
        waitForText("clicked", "fading");
    }

    /** The element's computed opacity, which isDisplayed collapses into a yes or no. */
    private double opacityOf(String id) {
        Object value = ((JavascriptExecutor) driver).executeScript(
                "return window.getComputedStyle(arguments[0]).opacity;", find(id));
        return Double.parseDouble(String.valueOf(value));
    }

    /** One measurement from the element's own box, in fractional pixels. */
    private double boxOf(String id, String property) {
        Object value = ((JavascriptExecutor) driver).executeScript(
                "return arguments[0].getBoundingClientRect()[arguments[1]];", find(id), property);
        return ((Number) value).doubleValue();
    }

    /**
     * Waits until an element's position stops changing between polls.
     *
     * <p>Measured in fractional pixels on purpose: {@code getRect} rounds to
     * whole ones, and a slow slide sits on the same rounded value for several
     * reads while still moving.
     */
    private void waitUntilStill(String id) {
        double[] previous = { Double.NaN };
        wait.until((ExpectedCondition<Boolean>) d -> {
            double now = boxOf(id, "top");
            boolean still = now == previous[0];
            previous[0] = now;
            return still;
        });
    }
}
