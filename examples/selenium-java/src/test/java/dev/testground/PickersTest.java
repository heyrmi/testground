package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.Keys;
import org.openqa.selenium.WebDriverException;
import org.openqa.selenium.WebElement;

/** /classic/pickers — the controls with nothing to type into, and what setting a value skips. */
class PickersTest extends Playground {

    private static final String PAGE = "/classic/pickers";

    @Test
    void theSliderMovesWithTheKeyboardWhichIsWhatAPersonWouldDo() {
        open(PAGE);
        WebElement slider = find("field-volume");

        assertEquals("30", slider.getDomProperty("value"));
        // sendKeys focuses the element first, so there is no separate focus
        // step; the arrows then move it by one step each.
        slider.sendKeys(Keys.ARROW_RIGHT, Keys.ARROW_RIGHT);

        // step=10, so two presses move it twenty.
        assertEquals("50", find("field-volume").getDomProperty("value"));

        clickAndWaitForReload("submit");
        waitForText("result-volume", "50");
    }

    @Test
    void settingTheSliderValueFromScriptSkipsTheEventsARealMoveFires() {
        open(PAGE);

        // The approach that looks right and is not. A slider has nothing to
        // type into, so assigning .value is the obvious move -- and it produces
        // the right number in the request, which is why it survives review.
        // What it does not produce is an input event. This zone runs no script,
        // so nothing notices; on any page with a listener behind the slider the
        // test would pass while the feature stayed broken.
        script("window.inputEvents = 0;"
                + "arguments[0].addEventListener('input', () => window.inputEvents++);",
                find("field-volume"));

        script("arguments[0].value = '70';", find("field-volume"));
        assertEquals(0L, script("return window.inputEvents;"));

        find("field-volume").sendKeys(Keys.ARROW_RIGHT);
        assertEquals(1L, script("return window.inputEvents;"));

        clickAndWaitForReload("submit");
        waitForText("result-volume", "80");
    }

    @Test
    void theColourInputCanOnlyBeSetNeverClickedThrough() {
        open(PAGE);
        WebElement colour = find("field-colour");
        assertEquals("#b4541e", colour.getDomProperty("value"));

        // Clicking opens an operating-system dialog that lives outside the page,
        // so no driver reaches it -- the same wall Playwright hits. What gets
        // past it is not typing: the WebDriver specification calls a colour
        // input a "non-typeable form control" and routes send keys on one
        // straight to a value assignment, so this line never presses a key at
        // all. It reads like the strongest possible interaction and is the
        // weakest, which is worth knowing before trusting it as coverage of the
        // picker.
        colour.sendKeys("#2f7d4f");
        assertEquals("#2f7d4f", find("field-colour").getDomProperty("value"));

        clickAndWaitForReload("submit");
        waitForText("result-colour", "#2f7d4f");
    }

    @Test
    void dateInputsPostTheFormatTheSpecificationFixesNotTheOneDisplayed() {
        open(PAGE);

        // Typing is deliberately not used here, and the reason is the lesson.
        // The specification calls a date input typeable, so sendKeys really does
        // press keys -- into a row of segments laid out in the browser's locale
        // order, treating the hyphens as nothing. On this machine
        // sendKeys("2026-03-14") leaves the field reading 60314-02-20 and the
        // time and month fields empty, which is enough to block the submit
        // entirely. A suite that typed here would pass or fail on how the
        // machine running it formats dates. The value is the portable unit: the
        // HTML specification fixes it as yyyy-mm-dd whatever gets rendered.
        setFromScript("field-date", "2026-03-14");
        setFromScript("field-time", "09:30");
        setFromScript("field-moment", "2026-03-14T09:30");
        setFromScript("field-month", "2026-03");
        setFromScript("field-week", "2026-W11");

        // An input has no text; whatever the segments show is browser chrome
        // the DOM never exposes. Reading the displayed date is not on offer.
        assertEquals("", find("field-date").getText());

        clickAndWaitForReload("submit");
        waitForPresent("result");
        assertEquals("2026-03-14", text("result-date"));
        assertEquals("09:30", text("result-time"));
        assertEquals("2026-03-14T09:30", text("result-moment"));
        assertEquals("2026-03", text("result-month"));
        assertEquals("2026-W11", text("result-week"));
    }

    @Test
    void aDateOutsideTheAllowedRangeFailsValidationBeforePosting() {
        open(PAGE);

        // Post something in range first, so the page carries a result the
        // blocked submit would have had to overwrite. The browser refuses an
        // invalid submit itself: no request, no error on the page, nothing to
        // wait for -- only the absence of a change, and an absence is not
        // observable unless something would otherwise have moved.
        setFromScript("field-deadline", "2026-06-15");
        clickAndWaitForReload("submit");
        waitForText("result-deadline", "2026-06-15");

        // A plain click on purpose: the helper used above waits for the document
        // to be replaced, and here nothing is going to replace it.
        setFromScript("field-deadline", "2020-01-01");
        click("submit");

        // Reading the field's validity is also the proof that the page did not
        // reload: a submit that got through would have served a fresh, empty
        // deadline field, and an empty one underflows nothing.
        assertTrue((Boolean) script("return arguments[0].validity.rangeUnderflow;", find("field-deadline")));
        assertEquals("2026-06-15", text("result-deadline"));
    }

    @Test
    void aDateInsideTheRangePostsNormally() {
        open(PAGE);

        setFromScript("field-deadline", "2026-06-15");
        clickAndWaitForReload("submit");

        waitForText("result-deadline", "2026-06-15");
    }

    /**
     * Assigns a control's value from page script.
     *
     * <p>Only for the controls WebDriver genuinely cannot drive -- the colour
     * input and the native date family. Everywhere else this would be cheating,
     * because it skips the events the real interaction fires.
     */
    private void setFromScript(String id, String value) {
        script("arguments[0].value = arguments[1];", find(id), value);
    }

    private Object script(String source, Object... args) {
        return ((JavascriptExecutor) driver).executeScript(source, args);
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
