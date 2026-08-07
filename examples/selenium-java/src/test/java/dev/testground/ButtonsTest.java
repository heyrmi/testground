package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.List;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.WebDriverException;
import org.openqa.selenium.WebElement;

/** /classic/buttons — telling a control's role from its appearance, and reading what actually reached the server. */
class ButtonsTest extends Playground {

    private static final String PAGE = "/classic/buttons";

    @Test
    void twoSubmitsShareANameAndAreToldApartByTheirValue() {
        open(PAGE);

        // Both post `action`, so the element clicked is invisible in the request
        // and only the value that arrived distinguishes them. Asserting "I
        // clicked Publish" proves nothing; asserting the server saw "publish"
        // does.
        clickAndWaitForReload("submit-save");
        waitForText("result-action", "save");

        clickAndWaitForReload("submit-publish");
        waitForText("result-action", "publish");
    }

    @Test
    void theAnchorIsALinkNotAButtonAndPostsNothing() {
        open(PAGE);
        clickAndWaitForReload("submit-save");
        waitForText("submission-count", "1");

        // Playwright separates these two by role: getByRole('link') simply will
        // not match a <button>. Selenium has no role engine, so the equivalent
        // check is the tag itself plus the href only a link carries -- the same
        // question asked with the tools to hand.
        WebElement anchor = find("link-button");
        assertEquals("a", anchor.getTagName());
        assertTrue(anchor.getDomAttribute("href").endsWith(PAGE));

        clickAndWaitForReload(anchor);

        // A GET of the same page, so the count the server keeps is untouched.
        assertTrue(driver.getCurrentUrl().endsWith(PAGE));
        waitForText("submission-count", "1");
    }

    @Test
    void theInertButtonDoesNothingWhichLooksExactlyLikeAFailedClick() {
        open(PAGE);

        // type=button in a zone that runs no script. The click lands, the
        // browser reports success, and nothing happens -- indistinguishable
        // from a click that missed. Only the server's view settles it.
        click("inert");

        waitForPresent("no-submission");
        assertTrue(driver.getCurrentUrl().endsWith(PAGE));
    }

    @Test
    void theDisabledButtonIsDeclaredDisabledRatherThanClickedHopefully() {
        open(PAGE);

        assertFalse(find("disabled").isEnabled(), "the disabled submit reported itself as enabled");
        waitForPresent("no-submission");
    }

    @Test
    void clickingTheDisabledButtonSucceedsAndProvesNothing() {
        open(PAGE);

        // The approach that looks right and is not, and it is worse in Selenium
        // than in Playwright. Playwright's actionability check makes a click on
        // a disabled control fail loudly; WebDriver dispatches it, the browser
        // swallows it, and the call returns normally. A test written this way
        // passes whether the control is disabled, broken or missing its
        // handler. The assertion above is the one that carries meaning.
        find("disabled").click();

        waitForPresent("no-submission");
        assertEquals(0, count("result-action"));
    }

    @Test
    void resetClearsTheFieldWithoutPosting() {
        open(PAGE);

        find("field-draft").sendKeys("some work in progress");
        click("reset");

        // type=reset restores the value the server served, which happens to be
        // empty here -- it does not clear the field to empty by definition.
        assertEquals("", find("field-draft").getDomProperty("value"));
        waitForPresent("no-submission");
    }

    @Test
    void aClickLandingOnAChildElementStillActivatesTheButton() {
        open(PAGE);

        // The label sits in a child span, so a real user's click usually lands
        // on the child rather than the button. The event still bubbles to the
        // submit, and the server sees the button's value.
        List<WebElement> labels = find("submit-icon").findElements(By.tagName("span"));
        clickAndWaitForReload(labels.get(labels.size() - 1));

        waitForText("result-action", "icon");
    }

    /**
     * Clicks a control that replaces the document, and waits until it has.
     *
     * <p>Without this the next assertion races the reload. Playground#waitForText
     * locates an element and then reads it in a second round trip, so it can
     * find one in the outgoing document and read it after the incoming one has
     * arrived -- the stale-element failure this zone is about, landing in the
     * test instead of the product. Playwright's locators re-resolve and hide the
     * problem; in Selenium the synchronisation has to be written down.
     *
     * <p>ExpectedConditions.stalenessOf is the obvious way to write it and is
     * not quite enough: it treats only StaleElementReferenceException as the
     * signal, and a driver asked about an element exactly as the document is
     * being swapped answers with a raw protocol error instead. Both mean the
     * same thing -- the document that held this element has gone.
     */
    private void clickAndWaitForReload(String id) {
        clickAndWaitForReload(find(id));
    }

    private void clickAndWaitForReload(WebElement control) {
        WebElement outgoing = find("form");
        control.click();

        wait.until(d -> {
            try {
                outgoing.isDisplayed();
                return false;
            } catch (WebDriverException gone) {
                return true;
            }
        });
        // The old document has gone; this waits for the new one to finish
        // arriving, so the assertion after the call reads a settled page.
        wait.until(d -> "complete".equals(
                ((JavascriptExecutor) d).executeScript("return document.readyState")));
    }
}
