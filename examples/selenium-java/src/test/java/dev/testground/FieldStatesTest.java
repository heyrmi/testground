package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertNotEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.WebDriverException;
import org.openqa.selenium.WebElement;

/** /classic/field-states — readonly, disabled and aria-disabled look identical and agree about nothing. */
class FieldStatesTest extends Playground {

    private static final String PAGE = "/classic/field-states";

    @Test
    void theThreeUneditableLookingFieldsBehaveCompletelyDifferently() {
        open(PAGE);

        assertEquals("true", find("field-readonly").getDomProperty("readOnly"));
        assertFalse(find("field-disabled").isEnabled());
        assertEquals("true", find("field-aria-disabled").getDomAttribute("aria-disabled"));
    }

    @Test
    void readonlyTakesFocusAndDisabledDoesNot() {
        open(PAGE);

        // The visible difference between these two is nothing; the behavioural
        // difference starts at focus. A keyboard user reaches the readonly field
        // and can copy out of it, and never reaches the disabled one.
        click("field-readonly");
        assertEquals("field-readonly", focused());

        click("field-disabled");
        assertNotEquals("field-disabled", focused());
    }

    @Test
    void seleniumTypesIntoTheAriaDisabledFieldWithoutComplainingAtAll() {
        open(PAGE);
        WebElement ariaDisabled = find("field-aria-disabled");

        // The approach that looks right and is not, and the two frameworks fail
        // it in opposite directions. Playwright treats aria-disabled as disabled
        // and refuses to type, so the defect announces itself as a broken test.
        // WebDriver's "is element enabled" is defined on the disabled IDL
        // attribute alone, so Selenium reports this field enabled and types into
        // it happily -- a suite written this way goes green and never mentions
        // that the control tells assistive technology it is unavailable.
        assertTrue(ariaDisabled.isEnabled(), "WebDriver reads only the disabled attribute, not the ARIA one");
        ariaDisabled.clear();
        ariaDisabled.sendKeys("typed by a person");

        clickAndWaitForReload("submit");
        waitForText("result-aria-disabled", "typed by a person");

        // So the finding has to be made deliberately: nothing in the interaction
        // will make it for you. The markup is what needs fixing, not the test.
        open(PAGE);
        assertEquals("true", find("field-aria-disabled").getDomAttribute("aria-disabled"));
        assertFalse(find("field-aria-disabled").getDomProperty("disabled").equals("true"));
    }

    @Test
    void onlyTheDisabledFieldIsAbsentFromTheRequest() {
        open(PAGE);
        clickAndWaitForReload("submit");

        waitForPresent("result-arrived");
        String arrived = text("result-arrived");

        assertTrue(arrived.contains("readonly"), arrived);
        assertTrue(arrived.contains("ariaDisabled"), arrived);
        assertFalse(arrived.contains("locked"), "a disabled control is not part of the form: " + arrived);

        assertEquals("posted anyway", text("result-readonly"));
        assertEquals("", text("result-locked"));
    }

    @Test
    void threeOfTheFourFieldsHaveAnAccessibleNameButSeleniumCanOnlyAskAfterFindingThem() {
        open(PAGE);

        // Playwright locates by accessible name: getByLabel finds the field, so
        // a nameless field simply cannot be found and the defect surfaces as a
        // failing locator. Selenium has no such locator -- getAccessibleName is
        // a question you ask an element you already hold, so the test id gets you
        // there first and the name is checked afterwards. Same finding, opposite
        // order, and the Selenium version only fails if you remember to ask.
        // Trimmed on purpose: the wrapping label contributes the whitespace
        // around the input to the computed name, so this one arrives as
        // "Labelled by wrapping ". Playwright's getByLabel normalises that for
        // you; asking the browser directly hands back exactly what it computed.
        assertEquals("Labelled with for", find("field-labelled-for").getAccessibleName().trim());
        assertEquals("Labelled by wrapping", find("field-labelled-wrap").getAccessibleName().trim());
        assertEquals("Labelled with aria-label", find("field-labelled-aria").getAccessibleName().trim());
    }

    @Test
    void thePlaceholderOnlyFieldIsNamedByNothingInTheMarkup() {
        open(PAGE);

        // A placeholder is not a label: it has no `for`, it names nothing, and
        // it vanishes the moment anyone types. Chrome's accessibility tree
        // papers over that by falling back to the placeholder when there is no
        // label at all, which is why getAccessibleName is not enough on its own
        // -- it reports a name the markup never gave. The markup is the question
        // worth asking.
        boolean named = (Boolean) ((JavascriptExecutor) driver).executeScript(
                "const el = arguments[0];"
                        + "return Boolean(el.getAttribute('aria-label')) || el.labels.length > 0;",
                find("field-unlabelled"));

        assertFalse(named, "no label element and no aria-label leaves the field nameless");
    }

    /** The test id of whatever currently has focus, or an empty string if nothing tagged does. */
    private String focused() {
        String id = driver.switchTo().activeElement().getDomAttribute("data-testid");
        return id == null ? "" : id;
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
