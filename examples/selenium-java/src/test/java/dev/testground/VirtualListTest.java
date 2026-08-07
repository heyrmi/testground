package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.List;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.StaleElementReferenceException;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.support.ui.ExpectedCondition;

/** /app/virtual-list — ten thousand rows, twenty elements, and one inner scroll container. */
class VirtualListTest extends Playground {

    /** Fixed by the page so the offset of row N is exactly N times this. */
    private static final int ROW_HEIGHT = 40;

    private static final int LAST = 9_999;

    @Test
    void tenThousandRowsExistButOnlyAWindowOfThemIsRendered() {
        open("/app/virtual-list");
        waitForText("row-total", "10000");

        int rendered = count("row");
        assertTrue(rendered > 0, "nothing was rendered at all");
        assertTrue(rendered < 50, "the list is windowed, not merely long; " + rendered + " rows is too many");

        // Counting nodes measures the window. The declared total is the only
        // number that is about the data.
        assertEquals(String.valueOf(rendered), text("row-rendered"));
    }

    @Test
    void aRowFarDownTheListIsSimplyNotThereYet() {
        open("/app/virtual-list");
        waitForText("row-total", "10000");

        // Not hidden, not off screen: never rendered. A wait would spend the
        // full timeout proving it, so this is an immediate count.
        assertEquals(0, driver.findElements(rowAt(LAST)).size());
    }

    @Test
    void scrollTheContainerNotTheWindowAndComputeTheOffset() {
        open("/app/virtual-list");
        waitForText("row-total", "10000");

        // One jump rather than a scroll loop: the rows are a fixed height and
        // positioned by index, which the page publishes so this arithmetic is
        // part of the contract rather than a guess.
        assertEquals(String.valueOf(ROW_HEIGHT), text("row-height"));
        scrollViewportTo(LAST * ROW_HEIGHT);

        waitForRow(LAST);
        assertEquals("09999", find(rowAt(LAST)).findElement(testId("row-index")).getText().trim());
    }

    @Test
    void scrollingTheWindowAchievesNothing() {
        open("/app/virtual-list");
        waitForText("row-total", "10000");

        // The document does not scroll; the container inside it does. This is
        // the step that looks like it worked -- no error, no exception, and the
        // list exactly where it was.
        ((JavascriptExecutor) driver).executeScript("window.scrollBy(0, 20000);");

        assertEquals(0, driver.findElements(rowAt(LAST)).size());
    }

    @Test
    void aRowScrolledOutOfViewDetachesUnderneathAHeldReference() {
        open("/app/virtual-list");
        waitForText("row-total", "10000");

        // This is a lesson the Playwright suite has no way to teach. A locator
        // there is a recipe that is re-resolved on every use, so a row being
        // recycled underneath it is invisible. A WebElement is a handle to one
        // node, and the virtualiser destroys that node the moment it leaves the
        // window.
        WebElement held = find("row");
        assertEquals("00000", held.findElement(testId("row-index")).getText().trim());

        scrollViewportTo(LAST * ROW_HEIGHT);
        waitForRow(LAST);

        assertThrows(
                StaleElementReferenceException.class,
                held::getText,
                "the row was recycled, so the reference to it cannot still be good");
    }

    @Test
    void theWholeDataSetIsAvailableWithoutScrollingForIt() {
        open("/app/virtual-list");

        // Navigating to the endpoint would work and would drag a megabyte of
        // JSON back through the driver as page source. Reading it in the page
        // and returning three numbers is the same request and a great deal less
        // of it.
        driver.manage().timeouts().scriptTimeout(TIMEOUT);
        @SuppressWarnings("unchecked")
        List<Object> summary = (List<Object>) ((JavascriptExecutor) driver).executeAsyncScript(
                "const done = arguments[arguments.length - 1];"
                        + "fetch('/api/app/virtual-list/rows').then(r => r.json())"
                        + "  .then(b => done([b.count, b.rows.length, b.rows[9999].index]))"
                        + "  .catch(e => done([String(e)]));");

        assertEquals(List.of(10_000L, 10_000L, 9_999L), summary);
    }

    @Test
    void theRowCountIsUnderTheCallerControl() {
        open("/app/virtual-list?count=25");
        waitForText("row-total", "25");

        // Twenty-five rows still overflow a container that holds about ten, so
        // the last one is windowed out until the container is scrolled. Short
        // lists are not a way around the virtualiser.
        assertEquals(0, driver.findElements(rowAt(24)).size());

        scrollViewportTo(24 * ROW_HEIGHT);
        waitForRow(24);
    }

    /** One rendered row, narrowed by the index the challenge publishes on it. */
    private static By rowAt(int index) {
        return By.cssSelector("[data-testid='row'][data-index='" + index + "']");
    }

    private void waitForRow(int index) {
        wait.until((ExpectedCondition<Boolean>) d -> !d.findElements(rowAt(index)).isEmpty());
    }

    /**
     * Moves the inner scroll container.
     *
     * <p>Assigning scrollTop is the equivalent of Playwright evaluating on the
     * element. A key press or a wheel gesture would land on whatever holds the
     * focus or the pointer, which on this page is usually not the container.
     */
    private void scrollViewportTo(int offset) {
        ((JavascriptExecutor) driver).executeScript("arguments[0].scrollTop = arguments[1];", find("viewport"), offset);
    }
}
