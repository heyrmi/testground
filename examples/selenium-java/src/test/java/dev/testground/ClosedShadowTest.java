package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.ElementNotInteractableException;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.NoSuchShadowRootException;
import org.openqa.selenium.SearchContext;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.interactions.Actions;

/** /wc/closed-shadow — a component the page cannot reach into, and the surface it offers instead. */
class ClosedShadowTest extends Playground {

    private static final String PAGE = "/wc/closed-shadow";

    @Test
    void asFarAsThePageIsConcernedThereIsNothingThere() {
        open(PAGE);

        // This is the absence the component was built to create, and it is what
        // any code running inside the page sees: no root, so no traversal, so
        // no elements.
        Object root = ((JavascriptExecutor) driver)
                .executeScript("return arguments[0].shadowRoot;", find("closed-host"));
        assertNull(root, "shadowRoot was not null, so the root was not closed");

        // The elements inside do carry test ids -- closed-input and
        // closed-submit are right there in the component's markup -- and a
        // page-wide search still returns nothing, with no indication that
        // anything is being withheld. That silence is the trap: it is
        // indistinguishable from a component that failed to load.
        assertEquals(0, count("closed-input"));
        assertEquals(0, count("closed-submit"));
        assertEquals(0, count("closed-echo"));

        // Being more specific does not help. A descendant combinator is one
        // query against one tree, and a shadow boundary is not a tree edge.
        assertEquals(
                0,
                driver.findElements(By.cssSelector("pg-closed-box [data-testid='closed-input']")).size(),
                "a descendant combinator crossed a closed boundary, which it must not");
    }

    @Test
    void anOpenRootBesideItIsReachableSoThePageIsNotSimplyBroken() {
        open(PAGE);

        // The control for the test above. Without it the honest reading of all
        // those zeroes is "the bundle never ran", and telling that apart from
        // "one root is closed" is most of the skill this page teaches.
        assertTrue(partButton().isDisplayed());
    }

    @Test
    void theDriverCanReachInsideAndThatIsNotPermissionToDoIt() {
        open(PAGE);

        // The trap, and it is a sharper one here than in the Playwright suite.
        //
        // Playwright evaluates in the page, where shadowRoot is null, so its
        // spec for this challenge can only record the absence above. WebDriver
        // sits outside the page: ChromeDriver answers Get Element Shadow Root
        // for a closed host anyway, and everything inside becomes locatable.
        // The component's own source says a closed root was never a security
        // boundary. This is what that means in practice.
        SearchContext smuggled = find("closed-host").getShadowRoot();
        WebElement input = smuggled.findElement(testId("closed-input"));
        assertTrue(input.isDisplayed());
        assertEquals("unreachable from the page", input.getDomAttribute("placeholder"));

        // And here the smuggled route stops behaving like a real one. Element
        // Send Keys checks the element can be focused, and that check reasons
        // about the document -- which a closed root's contents are never part
        // of. Locating worked, reading worked, clicking works, typing is
        // refused. The assertion pins that on purpose: if a later ChromeDriver
        // changes its mind, this failing is the notice.
        assertThrows(ElementNotInteractableException.class, () -> input.sendKeys("typed"));

        // The gap is not even self-consistent. Clicking focuses the field for
        // real, so driving the keyboard instead of the element types into it
        // perfectly well, and the component's own property agrees afterwards.
        input.click();
        new Actions(driver).sendKeys("reached in through the driver").perform();
        click("closed-read");
        waitForText("closed-value", "reached in through the driver");

        // So it works, mostly, sometimes. It is still the wrong test to write,
        // and the reason is not that it fails today. The author declared this
        // surface private, so the markup and the wiring inside carry no
        // promise and the next refactor breaks a suite that never touched the
        // component's contract. It is also a privilege of one driver on one
        // browser: anything running in the page -- the whole Playwright suite
        // next door, and every user of the component -- genuinely cannot do
        // this. A test that passes only because the runner has more authority
        // than the application is testing the runner.
        //
        // Everything below drives the same page through what it published.
    }

    @Test
    void thePropertyIsTheSupportedWayIn() {
        open(PAGE);

        click("closed-read");
        waitForText("closed-value", "(empty)");

        click("closed-write");
        waitForText("closed-value", "written through the property");
    }

    @Test
    void aTestCanDriveTheComponentThroughItsPropertyDirectly() {
        open(PAGE);

        // No traversal, because a caller is not entitled to one. This is the
        // whole supported surface, and a component with a closed root and no
        // such surface is one the test has correctly found a defect in: the
        // finding is about the component, not about the tooling.
        ((JavascriptExecutor) driver).executeScript(
                "arguments[0].value = 'set from the test';", find("closed-host"));

        click("closed-read");
        waitForText("closed-value", "set from the test");
    }

    @Test
    void aComposedEventStillCrossesTheClosedBoundary() {
        open(PAGE);

        // Closed stops the tree being walked; it does not stop an event
        // dispatched with composed: true. The light DOM is listening, so the
        // value lands somewhere assertable without the test ever having seen
        // the element that produced it.
        ((JavascriptExecutor) driver).executeScript(
                "arguments[0].value = 'escaping';"
                        + "arguments[0].dispatchEvent(new CustomEvent('pg-closed-submit', {"
                        + "  detail: { value: arguments[0].value }, bubbles: true, composed: true }));",
                find("closed-host"));

        waitForText("closed-escaped", "escaping");
    }

    @Test
    void theLateElementIsPresentAllAlongAndDoesNothing() {
        open(PAGE);

        // Waiting for the element succeeds immediately, which is the trap in
        // full: an unknown tag is still an element. It is attached, it has no
        // shadow root and no behaviour, and every assertion about its existence
        // passes while the component is not there.
        WebElement host = find("late-host");
        assertNull(host.getDomAttribute("data-upgraded"), "the element had already upgraded");
        assertThrows(NoSuchShadowRootException.class, host::getShadowRoot);
        assertEquals(0, count("late-content"));

        // The marker is what actually waits for the component. A custom element
        // upgrade is not a load event, so nothing in the navigation the driver
        // already waited on covers it.
        wait.until(d -> "true".equals(find("late-host").getDomAttribute("data-upgraded")));

        SearchContext upgraded = find("late-host").getShadowRoot();
        assertTrue(upgraded.findElement(testId("late-content")).getText().contains("Upgraded"));
    }

    @Test
    void partStylingReachesInWhereSelectorsCannot() {
        open(PAGE);

        // The page's own stylesheet set this through pg-part-box::part(trigger),
        // with no access to the element's internals and no knowledge of what
        // tag the part sits on. A hook the component chose to expose is what
        // should stand in for the reaching it is refusing.
        assertEquals("600", partButton().getCssValue("font-weight"));
    }

    /**
     * The button inside the part-exposing component's open root.
     *
     * <p>Walked rather than located page-wide, because WebDriver's CSS is
     * document.querySelectorAll underneath and that stops at every shadow
     * boundary. The Playwright suite finds this button with a plain locator;
     * piercing is a framework feature, not a browser one.
     */
    private WebElement partButton() {
        WebElement host = find("part-host");
        SearchContext root = wait.until(d -> {
            try {
                return host.getShadowRoot();
            } catch (NoSuchShadowRootException notUpgradedYet) {
                // The root is attached when the element upgrades, which is
                // after the module script has been fetched and run. Treating
                // this as an answer rather than as "not yet" is how a slow
                // bundle gets reported as a broken component.
                return null;
            }
        });
        return root.findElement(testId("part-button"));
    }
}
