package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.WebDriverException;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.support.ui.Select;

/** /classic/choices — repeated form fields, multi-selects and disabled options, read from the request rather than from the page. */
class ChoicesTest extends Playground {

    private static final String PAGE = "/classic/choices";

    @Test
    void oneCheckboxStartsCheckedAndTheGroupPostsEveryCheckedValue() {
        open(PAGE);

        assertTrue(find("topping-cheese").isSelected());
        assertFalse(find("topping-olives").isSelected());

        setChecked("topping-olives", true);
        clickAndWaitForReload("submit");

        // Three inputs share one name, so the request carries a repeated field
        // and the server joins what arrived. Order follows the document, not
        // the order they were ticked in.
        waitForText("result-toppings", "cheese, olives");
    }

    @Test
    void readingTheGroupValueByItsNameReportsTheFirstBoxNotTheCheckedOnes() {
        open(PAGE);

        setChecked("topping-cheese", false);
        setChecked("topping-anchovies", true);

        // The approach that looks right and is not. One name, so ask the form
        // for the value of that name -- except a repeated field has no single
        // value. This returns the value attribute of the first input carrying
        // the name, which is "cheese" even though cheese is now the only box
        // that is NOT ticked. The request is the only honest source.
        WebElement firstNamedTopping = driver.findElement(By.name("topping"));
        assertEquals("cheese", firstNamedTopping.getDomProperty("value"));
        assertFalse(firstNamedTopping.isSelected());

        clickAndWaitForReload("submit");
        waitForText("result-toppings", "anchovies");
    }

    @Test
    void anUncheckedBoxPostsNothingAtAllRatherThanAFalseValue() {
        open(PAGE);
        clickAndWaitForReload("submit");

        waitForText("result-toppings", "cheese");
        // Not "no", not "false", not "off": an unchecked checkbox contributes no
        // field to the body at all, so the server never learns it existed.
        assertEquals("", text("result-newsletter"));
    }

    @Test
    void selectingOneRadioClearsTheRestOfItsGroup() {
        open(PAGE);

        setChecked("delivery-standard", true);
        setChecked("delivery-express", true);

        assertFalse(find("delivery-standard").isSelected());
        clickAndWaitForReload("submit");
        waitForText("result-delivery", "express");
    }

    @Test
    void aMultiSelectTakesSeveralValuesThroughTheSelectApi() {
        open(PAGE);

        // Clicking the options twice would not do this: on a multi-select a
        // plain click replaces the selection unless the platform's modifier key
        // is held, and which key that is differs per platform. Selenium's Select
        // wrapper is the counterpart to Playwright's selectOption([...]) -- on a
        // multiple select each selectByValue adds to the selection rather than
        // replacing it.
        Select languages = new Select(find("field-languages"));
        assertTrue(languages.isMultiple());
        languages.selectByValue("ja");
        languages.selectByValue("sw");

        clickAndWaitForReload("submit");
        waitForText("result-languages", "ja, sw");
    }

    @Test
    void aDisabledOptionIsDeclaredDisabledRatherThanDiscoveredByClicking() {
        open(PAGE);

        // Options carry no test ids of their own -- the select is the published
        // contract and the option is addressed by the value it posts, which is
        // as stable as a test id would be.
        WebElement country = find("field-country");
        assertFalse(
                country.findElement(By.cssSelector("option[value='is']")).isEnabled(),
                "the option marked disabled reported itself as selectable");
        assertEquals(2, country.findElements(By.tagName("optgroup")).size());
    }

    @Test
    void optionsInsideAnOptgroupAreStillSelectableByValue() {
        open(PAGE);

        // An optgroup is a heading, not a level of nesting to walk into: the
        // option is still a direct child of the select as far as selecting goes.
        new Select(find("field-country")).selectByValue("jp");

        clickAndWaitForReload("submit");
        waitForText("result-country", "jp");
    }

    /**
     * Sets a checkbox or radio to a state rather than toggling it.
     *
     * <p>Playwright has check()/uncheck(), which are idempotent; WebDriver only
     * has click(), which toggles. Calling click() to "tick" a box that already
     * starts ticked -- as topping-cheese does -- silently unticks it, and the
     * assertion that fails is three lines later on the server's view.
     */
    private void setChecked(String id, boolean checked) {
        WebElement box = find(id);
        if (box.isSelected() != checked) {
            box.click();
        }
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
