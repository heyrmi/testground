package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertThrows;

import java.net.URI;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.Keys;
import org.openqa.selenium.StaleElementReferenceException;
import org.openqa.selenium.WebDriverException;
import org.openqa.selenium.WebElement;

/** /classic/text-inputs — post-redirect-get, and why every element handle held across it is dead. */
class TextInputsTest extends Playground {

    private static final String PAGE = "/classic/text-inputs";

    private static final String TEXT = "hello";
    private static final String PASSWORD = "hunter22";
    private static final String EMAIL = "name@example.test";
    private static final String COMMENT = "a comment";

    @Test
    void theFormPostsAndTheServerEchoesWhatItReceived() {
        open(PAGE);
        waitForPresent("no-submission");

        fillEverything();
        clickAndWaitForReload("submit");

        waitForPresent("result");
        assertEquals(TEXT, text("result-text"));
        assertEquals(EMAIL, text("result-email"));
        assertEquals(COMMENT, text("result-comment"));
        assertEquals("1", text("submission-count"));
    }

    @Test
    void thePasswordIsNeverReflectedBackIntoThePage() {
        open(PAGE);

        find("field-password").sendKeys(PASSWORD);
        clickAndWaitForReload("submit");

        waitForText("result-password", "8 characters, not echoed");
        assertFalse(
                driver.getPageSource().contains(PASSWORD),
                "the submitted password came back in the response body");
    }

    @Test
    void theRedirectIsWhatMakesARefreshSafe() {
        open(PAGE);
        clickAndWaitForReload("submit");
        waitForText("submission-count", "1");

        // The Playwright suite watches the POST return 303. WebDriver cannot see
        // a response status without attaching CDP, so the property is proved
        // from the outside instead, and arguably more usefully: the document on
        // screen came from the GET the redirect pointed at, so reloading repeats
        // that GET rather than the post. Without the 303 this would resubmit and
        // the count would climb.
        driver.navigate().refresh();

        waitForText("submission-count", "1");
        assertEquals(PAGE, URI.create(driver.getCurrentUrl()).getPath());
    }

    @Test
    void everyHandleHeldAcrossTheSubmitIsStaleBecauseSeleniumHasNoLazyLocator() {
        open(PAGE);

        WebElement before = find("field-text");
        clickAndWaitForReload("submit");
        waitForPresent("result");

        // Playwright can hold a locator across the reload because a locator is a
        // query re-resolved on use; only its element handles go stale. Selenium
        // has no such thing -- a WebElement is always a handle into one
        // document, so the discipline is to re-locate rather than to cache. This
        // is the single most common cause of a flaky Selenium suite, and the
        // helpers on Playground exist to make re-locating the path of least
        // effort.
        assertThrows(StaleElementReferenceException.class, () -> before.sendKeys("again"));

        find("field-text").sendKeys("again");
        assertEquals("again", find("field-text").getDomProperty("value"));
    }

    @Test
    void enterInATextFieldSubmitsWithoutTouchingTheButton() {
        open(PAGE);

        WebElement form = find("form");
        WebElement text = find("field-text");
        text.sendKeys("submitted with the keyboard");
        text.sendKeys(Keys.ENTER);

        // Enter posts the form, so the same reload has to be waited out as if
        // the button had been pressed -- nothing about the keyboard route makes
        // it synchronous.
        waitForDocumentSwap(form);
        waitForText("result-text", "submitted with the keyboard");
    }

    @Test
    void submissionsAccumulatePerSessionAndCanBeDiscarded() {
        open(PAGE);

        // Waiting between the two clicks is not politeness: the second click
        // would otherwise land on the button of a document already being
        // replaced, and WebDriver would report a stale element rather than a
        // missed submission.
        clickAndWaitForReload("submit");
        waitForText("submission-count", "1");
        clickAndWaitForReload("submit");
        waitForText("submission-count", "2");

        clickAndWaitForReload("clear");
        waitForPresent("no-submission");
        assertEquals(0, count("result"));
    }

    @Test
    void theNumberFieldCarriesItsOwnConstraints() {
        open(PAGE);
        WebElement number = find("field-number");

        assertEquals("0", number.getDomAttribute("min"));
        assertEquals("100", number.getDomAttribute("max"));
        assertEquals("5", number.getDomAttribute("step"));
    }

    @Test
    void settingAValueFromScriptLooksLikeAFasterFillAndDefeatsMaxlength() {
        open(PAGE);
        String tooLong = "a".repeat(250);

        // Typing goes through the browser's editing pipeline, so maxlength=200
        // truncates it exactly as it would for a user.
        find("field-comment").sendKeys(tooLong);
        assertEquals(200, find("field-comment").getDomProperty("value").length());

        // The approach that looks right and is not. Assigning .value from script
        // is tempting when sendKeys is slow, but it is not typing: it skips the
        // constraint the page relies on, and here it posts 250 characters
        // through a field the product believes cannot hold more than 200. On a
        // scripted page it would also skip every input event a framework is
        // listening for.
        ((JavascriptExecutor) driver).executeScript(
                "arguments[0].value = arguments[1];", find("field-comment"), tooLong);
        clickAndWaitForReload("submit");

        waitForPresent("result-comment");
        assertEquals(250, text("result-comment").length());
    }

    /** Fills every text-flavoured field; the fields start empty, so sendKeys appending is harmless. */
    private void fillEverything() {
        find("field-text").sendKeys(TEXT);
        find("field-password").sendKeys(PASSWORD);
        find("field-email").sendKeys(EMAIL);
        find("field-number").sendKeys("25");
        find("field-tel").sendKeys("+44 20 7946 0000");
        find("field-url").sendKeys("https://example.test");
        find("field-search").sendKeys("needle");
        find("field-comment").sendKeys(COMMENT);
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
        waitForDocumentSwap(outgoing);
    }

    private void waitForDocumentSwap(WebElement outgoing) {
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
