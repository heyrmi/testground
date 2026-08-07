package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import java.util.List;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.StaleElementReferenceException;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.support.ui.ExpectedCondition;
import org.openqa.selenium.support.ui.WebDriverWait;

/** /app/detached-elements — a list that rebuilds under you, and what survives it. */
class DetachedElementsTest extends Playground {

    private static final String PAGE = "/app/detached-elements";

    /**
     * The trap, and in Selenium it has a name: a {@link WebElement} is a
     * reference to one node, not a recipe for finding it again.
     */
    @Test
    void aReferenceTakenBeforeARebuildIsDetachedAfterIt() {
        open(PAGE + "?churnMs=200");

        WebElement charlie = driver.findElement(row("charlie"));
        click("toggle-churn");
        assertEquals("true", find("toggle-churn").getAttribute("data-churning"));
        waitForGenerationToAdvance();

        // A row for charlie is plainly on screen. This is not that row: React
        // discarded the node and built another, so touching the old reference
        // fails with a message about a stale element -- which names the symptom
        // and not the rebuild that caused it.
        assertThrows(StaleElementReferenceException.class, charlie::getText);
        assertEquals(1, driver.findElements(row("charlie")).size());
    }

    @Test
    void reFindingOnEveryUseSurvivesWhatAReferenceDoesNot() {
        open(PAGE + "?churnMs=200");
        click("toggle-churn");
        waitForGenerationToAdvance();

        // Playwright's locator re-resolves itself on every use, so a rebuild is
        // simply not its problem. Selenium has no such thing: a By is only a
        // query, and the element it returns can be discarded in the moment
        // between finding it and clicking it. Re-finding *and* retrying the
        // stale click is what buys the same immunity.
        clickThroughTheChurn(By.cssSelector(
                "[data-testid='unstable-row'][data-name='delta'] [data-testid='row-action']"));

        waitForTextContaining("chosen", "delta");
    }

    @Test
    void theDomIdsAreCorrectUntilTheNextTick() {
        open(PAGE + "?churnMs=200");

        String before = textThroughTheChurn("row-dom-id");
        click("toggle-churn");
        waitForGenerationToAdvance();

        // Even reading text has to tolerate the rebuild: the base class's
        // text() finds the element and then asks it, and the row can be
        // discarded in between those two round trips.
        String after = textThroughTheChurn("row-dom-id");
        assertNotEquals(before, after, "a selector built from the old id would still be pointing at it");
        assertEquals(
                0,
                driver.findElements(By.id(before)).size(),
                "the id from the previous generation still resolves, so the rebuild was not a rebuild");
    }

    @Test
    void theVanishingButtonHasToBeCaught() {
        open(PAGE);

        click("summon");
        // Six hundred milliseconds from appearing to gone. Nothing slow may
        // happen between deciding to click and clicking -- no screenshot, no
        // logging round trip, no second locate to "make sure".
        click("vanishing");
        waitForText("vanish-clicks", "1");

        waitForAbsent("vanishing");
    }

    @Test
    void aFieldCanUnmountWhileItIsBeingFilledIn() {
        open(PAGE);

        click("arm-unmount");
        WebElement field = find("doomed-field");
        field.sendKeys("half a sen");

        waitForPresent("form-gone");
        assertEquals(0, count("doomed-field"), "the field outlived the unmount it was armed for");

        // The rest of the sentence has nowhere to go. The reference was valid
        // when it was taken and the failure arrives at the keystroke, which is
        // why an unmount mid-interaction reads as a typing bug.
        assertThrows(StaleElementReferenceException.class, () -> field.sendKeys("tence"));
    }

    /** The row for a name, located by the attribute that survives the rebuild rather than by the id that does not. */
    private static By row(String name) {
        return By.cssSelector("[data-testid='unstable-row'][data-name='" + name + "']");
    }

    /** Waits for at least one rebuild, which is the signal that the churn is running. */
    private void waitForGenerationToAdvance() {
        wait.until((ExpectedCondition<Boolean>) d -> {
            List<WebElement> found = d.findElements(testId("generation"));
            return !found.isEmpty() && !found.get(0).getText().trim().equals("0");
        });
    }

    /**
     * Clicks something that may be replaced between the locate and the click.
     *
     * <p>The base class's {@code wait} does not ignore
     * {@link StaleElementReferenceException} -- it is not a {@code
     * NotFoundException} -- so a plain wait would fail on exactly the tick this
     * challenge exists to produce. Retrying the whole locate-and-click is the
     * Selenium spelling of an auto-retrying locator.
     */
    private void clickThroughTheChurn(By by) {
        new WebDriverWait(driver, TIMEOUT)
                .ignoring(StaleElementReferenceException.class)
                .until(d -> {
                    d.findElement(by).click();
                    return true;
                });
    }

    /** Reads text from an element that may be replaced between the locate and the read. */
    private String textThroughTheChurn(String id) {
        return new WebDriverWait(driver, TIMEOUT)
                .ignoring(StaleElementReferenceException.class)
                .until(d -> {
                    List<WebElement> found = d.findElements(testId(id));
                    return found.isEmpty() ? null : found.get(0).getText().trim();
                });
    }

    private void waitForTextContaining(String id, String fragment) {
        wait.until((ExpectedCondition<Boolean>) d -> {
            List<WebElement> found = d.findElements(testId(id));
            return !found.isEmpty() && found.get(0).getText().contains(fragment);
        });
    }
}
