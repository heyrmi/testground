package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.interactions.Actions;

/** /app/drag-and-drop — native drag events, pointer gestures, and how much of the answer belongs to the driver. */
class DragAndDropTest extends Playground {

    private static final String PAGE = "/app/drag-and-drop";

    @Test
    void nativeDragAndDropMovesAParcel() {
        open(PAGE);
        waitForText("delivered-count", "0");

        new Actions(driver).dragAndDrop(driver.findElement(parcel("crate")), find("dropzone")).perform();

        waitForText("delivered-count", "1");
        assertEquals(1, driver.findElements(delivered("crate")).size());
    }

    @Test
    void theListReordersByDroppingOneItemOntoAnother() {
        open(PAGE);
        waitForText("sortable-order", "one, two, three, four");

        new Actions(driver).dragAndDrop(sortableItem("four"), sortableItem("one")).perform();

        waitForText("sortable-order", "four, one, two, three");
    }

    /** The handle wants the opposite thing: real input, with at least one move between press and release. */
    @Test
    void thePointerHandleNeedsAPressAMoveAndARelease() {
        open(PAGE);
        WebElement rail = find("rail");
        int width = rail.getSize().getWidth();

        // Offsets are measured from the element's centre, and moving to an
        // element scrolls it into view first. Raw screen coordinates would not,
        // which is its own quiet way to miss.
        new Actions(driver)
                .moveToElement(rail, -(width / 2) + 4, 0)
                .clickAndHold()
                // A jump straight to the destination lands nowhere here: there
                // is no drop target, only a handler reading each move.
                .moveByOffset((int) (width * 0.6) - 4, 0)
                .release()
                .perform();

        int position = Integer.parseInt(text("handle-position"));
        assertTrue(position > 50 && position < 70, "the handle ended at " + position + ", so the moves were not seen");
    }

    /**
     * Whether pressing and moving the mouse counts as a drag is the driver's
     * answer, not the page's — and the two suites disagree about it.
     *
     * <p>The Playwright spec performs this exact gesture and asserts that
     * nothing is delivered, because its raw mouse events never turn on the
     * browser's drag machinery. The same gesture through chromedriver does
     * start a native drag, so the parcel arrives. Nothing about the page
     * changed. A Selenium suite that had copied Playwright's expectation here
     * would be asserting something false about this browser.
     */
    @Test
    void whetherAMouseSequenceCountsAsADragIsTheDriversAnswer() {
        open(PAGE);
        WebElement letter = driver.findElement(parcel("letter"));

        new Actions(driver)
                .moveToElement(letter)
                .clickAndHold()
                .moveToElement(find("dropzone"))
                .moveByOffset(3, 3)
                .release()
                .perform();

        waitForText("delivered-count", "1");
    }

    /** One technique covering two event families that share nothing. */
    @Test
    void oneRealInputTechniqueCoversBothFamilies() {
        open(PAGE);

        // dragAndDrop is press, move, release. The handle listens for pointer
        // events and has never heard of dragging; the parcels listen for drag
        // events and never see a pointer. Real input is what both are made of,
        // so the same command reaches both.
        new Actions(driver).dragAndDrop(find("handle"), find("rail")).perform();
        int position = Integer.parseInt(text("handle-position"));
        assertTrue(position > 40 && position < 60, "the handle should have followed the pointer to the middle of the rail, not to " + position);

        new Actions(driver).dragAndDrop(driver.findElement(parcel("parcel")), find("dropzone")).perform();
        waitForText("delivered-count", "1");
    }

    /**
     * The fallback every driver that cannot drag reaches for, and the trap in it.
     *
     * <p>Raising the events by hand is the documented workaround, and against the
     * drop zone it looks like a complete solution. It is not: the events all
     * land in one task, and anything that remembers what was picked up between
     * dragstart and drop has not been given the chance to.
     */
    @Test
    void dispatchingTheDragEventsYourselfIsAFallbackWithItsOwnTrap() {
        open(PAGE);

        // The drop zone reads the parcel straight out of the dataTransfer, so
        // four dispatched events are all it ever needed.
        dispatchDragInOneTask(driver.findElement(parcel("crate")), find("dropzone"));
        waitForText("delivered-count", "1");

        // The list is not so easy. It records what was picked up in component
        // state that dragstart sets, and the drop handler in this same task is
        // still the one rendered before the drag began -- so it sees nothing
        // held and returns without reordering. No error is raised anywhere.
        dispatchDragInOneTask(sortableItem("four"), sortableItem("one"));
        assertEquals(
                "one, two, three, four",
                text("sortable-order"),
                "the reorder went through in one task, so this trap no longer needs demonstrating");

        // A fresh page, because that failed attempt did leave the item held: a
        // second try on the same page would succeed for the wrong reason.
        open(PAGE);
        dispatchDragAcrossTasks(sortableItem("four"), sortableItem("one"));
        waitForText("sortable-order", "four, one, two, three");
    }

    /** Raises the whole drag synchronously, with one dataTransfer shared across the events. */
    private void dispatchDragInOneTask(WebElement source, WebElement target) {
        ((JavascriptExecutor) driver).executeScript(
                DRAG_PRELUDE
                        + "fire(source, 'dragstart');"
                        + "fire(target, 'dragover');"
                        + "fire(target, 'drop');"
                        + "fire(source, 'dragend');",
                source, target);
    }

    /** The same events, with the page given a turn in between — which is what a real drag gives it for free. */
    private void dispatchDragAcrossTasks(WebElement source, WebElement target) {
        ((JavascriptExecutor) driver).executeAsyncScript(
                DRAG_PRELUDE
                        + "const done = arguments[2];"
                        + "fire(source, 'dragstart');"
                        + "setTimeout(() => {"
                        + "  fire(target, 'dragover');"
                        + "  fire(target, 'drop');"
                        + "  fire(source, 'dragend');"
                        + "  done();"
                        + "}, 0);",
                source, target);
    }

    private static final String DRAG_PRELUDE =
            "const source = arguments[0], target = arguments[1];"
                    + "const transfer = new DataTransfer();"
                    + "const fire = (node, type) => node.dispatchEvent("
                    + "  new DragEvent(type, { bubbles: true, cancelable: true, dataTransfer: transfer }));";

    /** The manifest publishes data-name as the way to tell these apart; nothing else distinguishes them. */
    private static By parcel(String name) {
        return By.cssSelector("[data-testid='parcel'][data-name='" + name + "']");
    }

    private static By delivered(String name) {
        return By.cssSelector("[data-testid='delivered'][data-name='" + name + "']");
    }

    private WebElement sortableItem(String name) {
        return driver.findElement(
                By.cssSelector("[data-testid='sortable-item'][data-name='" + name + "']"));
    }
}
