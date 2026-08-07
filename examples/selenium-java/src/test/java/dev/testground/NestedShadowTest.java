package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.List;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.SearchContext;
import org.openqa.selenium.WebElement;

/** /wc/nested-shadow — three open shadow roots, and the traversal WebDriver makes you write. */
class NestedShadowTest extends Playground {

    private static final String PAGE = "/wc/nested-shadow";

    @Test
    void aPageWideSearchIsBlindToEveryShadowRoot() {
        open(PAGE);

        // Walk in first, so the zeroes below mean "unreachable" rather than
        // "not rendered yet". Asserting a count of nothing against a page that
        // has not finished booting proves nothing at all.
        SearchContext inner = innerRoot();
        assertTrue(inner.findElement(testId("inner-input")).isDisplayed());

        // This is where the two reference suites part company, and the
        // difference is the reason both exist. Playwright's locators pierce
        // open roots, so its spec for this page opens by finding the input as
        // if it sat in the light DOM. By.cssSelector is document.querySelectorAll
        // underneath, which stops dead at the first boundary -- so in Selenium
        // the very same call is the trap. Somebody arriving from the other tool
        // reads this nothing as "the component failed to render" and goes
        // looking for a bug in the page.
        assertEquals(0, count("inner-input"));
        assertEquals(0, count("inner-submit"));
        assertEquals(0, count("inner-echo"));

        // Nor does being more specific help. A descendant combinator is still
        // one query against one tree, and the boundary is not a tree edge.
        assertEquals(
                0,
                driver.findElements(By.cssSelector("pg-shadow-outer [data-testid='inner-input']")).size(),
                "a descendant combinator crossed a shadow boundary, which it must not");
    }

    @Test
    void enteringEachRootExplicitlyIsWhatReachesTheInput() {
        open(PAGE);

        // Every host has to be found inside the root above it, one step at a
        // time. Note what the steps are keyed on: the hosts three deep carry no
        // test id, so the only handle is the custom element name. That is the
        // component's published contract rather than a styling detail, which is
        // why it is not the CSS-selector shortcut it resembles.
        WebElement outerHost = find("shadow-host");
        SearchContext outerRoot = shadowRootOf(outerHost);

        WebElement middleHost = outerRoot.findElement(By.cssSelector("pg-shadow-middle"));
        SearchContext middleRoot = shadowRootOf(middleHost);

        WebElement innerHost = middleRoot.findElement(By.cssSelector("pg-shadow-inner"));
        SearchContext innerRoot = shadowRootOf(innerHost);

        // All three are open, which is the only reason any of this works. One
        // closed root anywhere in the chain ends the walk -- see ClosedShadowTest.
        assertTrue(innerRoot.findElement(testId("inner-input")).isDisplayed());
        assertTrue(innerRoot.findElement(testId("inner-submit")).isDisplayed());
    }

    @Test
    void aSlottedNodeIsRenderedInARootItDoesNotBelongTo() {
        open(PAGE);

        // Found by an ordinary page-wide search even though it appears inside
        // the outer root: projection moves where a node is painted, not where
        // it lives. The contrast with the input above is the whole lesson.
        WebElement label = find("slotted-label");
        assertTrue(label.isDisplayed());

        Object inLightDom = ((JavascriptExecutor) driver)
                .executeScript("return arguments[0].getRootNode() === document;", label);
        assertEquals(true, inLightDom, "the slotted node had been adopted into a shadow root");
    }

    @Test
    void aComposedEventEscapesEveryBoundary() {
        open(PAGE);
        SearchContext inner = innerRoot();

        inner.findElement(testId("inner-input")).sendKeys("crossed all three");
        inner.findElement(testId("inner-submit")).click();

        // The light-DOM echo needs no traversal at all. When a component offers
        // a composed event, asserting on what escaped is a far cheaper contract
        // than reaching in for what produced it -- and it keeps working if the
        // internals are rearranged.
        waitForText("shadow-echo", "crossed all three");
        waitForText("shadow-submit-count", "1");

        // The mirror three roots down agrees, so the escape is not hiding a
        // component that never updated itself.
        waitForShadowText(inner, "inner-echo", "crossed all three");
    }

    @Test
    void theEventIsWhatCrossesRatherThanTheValue() {
        open(PAGE);
        SearchContext inner = innerRoot();

        // Typing alone updates the component and nothing outside it. A test
        // that fills the field and then reads the light-DOM echo is asserting
        // on a submit that never happened.
        inner.findElement(testId("inner-input")).sendKeys("typed but not sent");
        waitForShadowText(inner, "inner-echo", "typed but not sent");
        assertNotEquals("typed but not sent", text("shadow-echo"));
        assertEquals("0", text("shadow-submit-count"));

        inner.findElement(testId("inner-submit")).click();
        waitForText("shadow-echo", "typed but not sent");
    }

    /** The innermost of the three roots, walked from the light DOM each time. */
    private SearchContext innerRoot() {
        SearchContext outer = shadowRootOf(find("shadow-host"));
        SearchContext middle = shadowRootOf(outer.findElement(By.cssSelector("pg-shadow-middle")));
        return shadowRootOf(middle.findElement(By.cssSelector("pg-shadow-inner")));
    }

    /**
     * A host's shadow root, once it has one.
     *
     * <p>The root is attached when the custom element upgrades, which is after
     * the module script has been fetched and run. Calling getShadowRoot before
     * that raises NoSuchShadowRootException, and treating that as an answer
     * rather than as "not yet" is how a slow bundle gets reported as a broken
     * component.
     */
    private SearchContext shadowRootOf(WebElement host) {
        return wait.until(d -> {
            try {
                return host.getShadowRoot();
            } catch (org.openqa.selenium.NoSuchShadowRootException notUpgradedYet) {
                return null;
            }
        });
    }

    /** waitForText, but scoped to a shadow root instead of the document. */
    private void waitForShadowText(SearchContext root, String id, String expected) {
        wait.until(d -> {
            List<WebElement> found = root.findElements(testId(id));
            return !found.isEmpty() && found.get(0).getText().trim().equals(expected);
        });
    }
}
