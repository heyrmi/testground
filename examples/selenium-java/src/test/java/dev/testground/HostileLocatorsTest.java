package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.List;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.WebElement;

/**
 * /app/hostile-locators — the one page here where the test ids are withheld on
 * purpose, so every locator in this class is one the markup forced.
 */
class HostileLocatorsTest extends Playground {

    private static final String PAGE = "/app/hostile-locators";

    /** A zero-width space: present in the DOM, absent from the screen. */
    private static final String ZERO_WIDTH = "​";

    @Test
    void aClassNameSelectorIsCorrectUntilTheNextDeploy() {
        open(PAGE);

        String before = text("sample-class");
        assertEquals(1, driver.findElements(By.cssSelector("." + before)).size());

        click("rebuild");

        // No code changed and nothing was renamed by hand: the class name is a
        // content hash and a deploy moved it. The selector is simply gone, and
        // the failure arrives with nothing anyone will connect to it.
        assertEquals(0, driver.findElements(By.cssSelector("." + before)).size());
        assertNotEquals(before, text("sample-class"));
    }

    @Test
    void twoElementsShareOneIdAndTheLookupsDisagree() {
        open(PAGE);

        // Invalid HTML the browser accepts in silence. Selenium's By.id is a CSS
        // id selector underneath, so it sees both; the DOM's own getElementById
        // sees one. Neither is wrong, which is the problem.
        assertEquals(2, waitForAtLeast(By.id("duplicate"), 2).size());
        assertEquals(
                1L,
                ((JavascriptExecutor) driver).executeScript(
                        "return document.getElementById('duplicate') ? 1 : 0;"));

        // findElement returns the first of the two. Whichever one your tool
        // picks, it picked -- and it will keep picking it until someone reorders
        // the markup.
        driver.findElement(By.id("duplicate")).click();
        waitForText("chosen", "first-duplicate");
    }

    @Test
    void textSplitAcrossNodesDefeatsAnExactMatch() {
        open(PAGE);

        // The user sees one sentence, and so does anything that reads the
        // subtree's text.
        assertEquals("Order number 4417", text("split-text"));

        // But no node holds that string, so the exact-match locator everyone
        // writes first finds nothing at all against text that is plainly on
        // screen.
        assertEquals(0, driver.findElements(By.xpath("//*[text()='Order number 4417']")).size());
        assertEquals(
                List.of("Order", "number", "4417"),
                find("split-text").findElements(By.tagName("span")).stream().map(WebElement::getText).toList());
    }

    @Test
    void invisibleCharactersSitBetweenTheWords() {
        open(PAGE);

        // getText() reports what is rendered and zero-width characters render as
        // nothing, so the DOM property is what tells the truth here.
        String raw = find("zero-width").getDomProperty("textContent");

        assertTrue(raw.contains(ZERO_WIDTH), "the invisible characters have gone, so this trap is no longer here");
        assertNotEquals("Total: 42", raw, "the obvious assertion fails against text that looks right");

        // Normalising before comparing is what makes this tractable.
        assertEquals("Total: 42", raw.replace(ZERO_WIDTH, ""));
    }

    @Test
    void whatAUserCanReadAndWhatATestCanReadHaveDiverged() {
        open(PAGE);
        WebElement truncated = find("truncated");

        // CSS is drawing an ellipsis over most of this sentence.
        assertEquals(
                true,
                ((JavascriptExecutor) driver).executeScript(
                        "return arguments[0].scrollWidth > arguments[0].clientWidth;", truncated));

        // And the test can read every word of it, including the ones no user
        // can. An assertion that "the page shows" this sentence would be true of
        // the DOM and false of the screen.
        assertTrue(truncated.getText().contains("longer than the box that is drawing it"));
    }

    @Test
    void identicalTwinsCanOnlyBeToldApartByPositionWhichIsTheFinding() {
        open(PAGE);

        // Selenium has no locator for an accessible name, so the closest thing
        // to Playwright's getByRole is the text on the button -- and it matches
        // both, because same text, same class, same everything.
        List<WebElement> twins = waitForAtLeast(By.xpath("//button[normalize-space()='Continue']"), 2);
        assertEquals(2, twins.size());

        // Nothing distinguishes them but order. Using that works, and is a note
        // to go and fix the markup rather than a technique to be pleased with:
        // the day someone adds a third button above them, this test clicks the
        // wrong one and still passes until the assertion below catches it.
        twins.get(1).click();
        waitForText("chosen", "twin-right");
    }

    @Test
    void theDivSoupHasNothingToLocateByButItsText() {
        open(PAGE);

        // Every wrapper contains the word, so a contains match finds the
        // outermost one -- twelve levels above the element that does anything.
        assertTrue(
                waitForAtLeast(By.xpath("//div[contains(., 'Approve')]"), 13).size() >= 13,
                "the wrappers have gone, so matching on subtree text is no longer ambiguous");

        // Only the leaf holds it as its own text node. That is the whole of the
        // available signal: no role, no label, no test id, no heading.
        List<WebElement> leaf = driver.findElements(By.xpath("//div[normalize-space(text())='Approve']"));
        assertEquals(1, leaf.size());
        assertTrue(
                driver.findElements(By.xpath("//div[normalize-space(text())='Approve']/ancestor::div")).size() >= 12,
                "twelve wrappers, none of them meaning anything");

        leaf.get(0).click();
        waitForText("chosen", "div-soup");
    }

    /**
     * Waits until a raw locator matches at least {@code atLeast} elements.
     *
     * <p>This page is React, so open() returns on the document and the markup
     * arrives afterwards. findElements answers with an empty list rather than
     * retrying, so asserting on its size straight after a navigation asserts on
     * whatever had rendered by then -- which is everything on a fast machine and
     * nothing on a loaded one. The test ids elsewhere in this class hide the
     * problem because find() already waits; the challenge withholds them here,
     * which is exactly where the waiting has to be done by hand.
     */
    private List<WebElement> waitForAtLeast(By by, int atLeast) {
        return wait.until(d -> {
            List<WebElement> found = d.findElements(by);
            return found.size() >= atLeast ? found : null;
        });
    }
}
