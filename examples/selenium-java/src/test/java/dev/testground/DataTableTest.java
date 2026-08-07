package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertNotEquals;

import java.util.ArrayList;
import java.util.List;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.Keys;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.support.ui.ExpectedCondition;

/** /app/data-table — server-side sorting, a checkbox with three states, and edits that commit on blur. */
class DataTableTest extends Playground {

    private static final String PAGE = "/app/data-table";

    @Test
    void sortingIsARoundTripSoTheOldRowsAreStillOnScreen() {
        open(PAGE);
        waitForCount("row", 10);

        String before = find("row").getAttribute("data-id");
        click("sort-name");

        // Reading the first row here is the mistake the challenge is built
        // around: the click has only asked the server for a new order, and the
        // rows underneath it are still the previous answer. The table publishes
        // the sort its rows were fetched with, so waiting for that waits for the
        // thing that actually changed.
        waitForText("current-sort", "name asc");

        String after = find("row").getAttribute("data-id");
        assertNotEquals(before, after, "the rows never changed, so the sort did not round trip");
    }

    @Test
    void clickingTheSameHeaderAgainReversesIt() {
        open(PAGE);

        click("sort-amount");
        waitForText("current-sort", "amount asc");

        click("sort-amount");
        waitForText("current-sort", "amount desc");
    }

    @Test
    void amountsSortNumericallyNotAsText() {
        open(PAGE + "?size=5");
        click("sort-amount");
        waitForText("current-sort", "amount asc");

        List<Double> amounts = amountsOnScreen();
        List<Double> ascending = new ArrayList<>(amounts);
        ascending.sort(null);
        // Sorted as text, 9.99 would come after 1000.00 and this comparison is
        // the only thing that notices.
        assertEquals(ascending, amounts, "the amount column was ordered as text rather than as money");
    }

    @Test
    void selectAllHasAThirdStateThatIsAPropertyNotAnAttribute() {
        open(PAGE);
        waitForCount("row", 10);

        click("select-row");
        waitForText("selected-count", "1");

        // Neither checked nor unchecked, and it appears nowhere in the markup:
        // getAttribute would find nothing to report. Selenium reaches the live
        // property with getDomProperty, which is the same reach as Playwright
        // evaluating el.indeterminate.
        WebElement all = find("select-all");
        assertFalse(all.isSelected(), "the box claimed every row was chosen when one was");
        assertEquals("true", all.getDomProperty("indeterminate"));

        all.click();
        assertEquals("false", find("select-all").getDomProperty("indeterminate"));
        waitForText("selected-count", "10");
    }

    @Test
    void aSelectionSurvivesPagingBecauseItIsNotStoredInTheRows() {
        open(PAGE);
        waitForCount("row", 10);

        click("select-all");
        waitForText("selected-count", "10");

        click("page-next");
        waitForText("page-label", "page 2 of 12");

        // The count is unchanged because the selection is kept beside the rows
        // rather than in them; the header box is clear because none of *these*
        // rows are in it.
        waitForText("selected-count", "10");
        assertFalse(find("select-all").isSelected());
    }

    @Test
    void anEditedCellIsNotSavedUntilFocusLeavesIt() {
        open(PAGE);
        waitForCount("row", 10);

        click("cell-note");
        fill("cell-note-input", "written but not committed");

        // Still in the input: nine cells are showing their committed value and
        // the tenth is the editor. Reading the cell now would read a value that
        // has been saved nowhere.
        assertEquals(9, count("cell-note"));

        // Playwright can call blur() on the element. WebDriver has no such
        // command, so focus is moved the way a person moves it -- and moving it
        // is precisely what the page is waiting for.
        find("cell-note-input").sendKeys(Keys.TAB);
        waitForAttribute("cell-note", "data-committed", "written but not committed");
    }

    @Test
    void emptyIsAnOutcomeNotASlowSuccess() {
        open(PAGE);
        fill("filter", "nobodyisnamedthis");

        waitForPresent("table-empty");
        waitForText("total-rows", "0");
        assertEquals(0, count("table"), "the previous rows were still on screen beside the empty message");
    }

    @Test
    void errorIsAnOutcomeTooAndSaysSo() {
        open(PAGE + "?state=error");

        waitForPresent("table-error");
        // A wait that only knows how to look for rows would have timed out here
        // and blamed the wait. These two assertions are what separate "refused"
        // from "nothing matched" and from "not yet".
        assertEquals(0, count("table-empty"));
        assertEquals(0, count("table-loading"));
    }

    @Test
    void theLoadingStateIsDistinguishableFromBothOfThem() {
        open(PAGE + "?state=slow");

        waitForPresent("table-loading");
        waitForPresent("table");
        assertEquals(0, count("table-loading"));
    }

    @Test
    void filteringResetsToTheFirstPage() {
        open(PAGE);
        click("page-next");
        waitForText("page-label", "page 2 of 12");

        // Without this, filtering from page two would ask for a page that the
        // narrowed result set may not have.
        fill("filter", "a");
        waitForTextContaining("page-label", "page 1 of");
    }

    /**
     * The amounts on screen, in the order they are displayed.
     *
     * <p>The money column carries no test id of its own -- only the row does --
     * so the cell is reached by position within a row that was located properly.
     * The cost is real: inserting a column ahead of it breaks this and nothing
     * else in the class.
     */
    private List<Double> amountsOnScreen() {
        List<Double> amounts = new ArrayList<>();
        for (WebElement row : findAll("row")) {
            amounts.add(Double.parseDouble(row.findElement(By.cssSelector("td:nth-child(4)")).getText().trim()));
        }
        return amounts;
    }

    /**
     * Replaces a field's whole value by typing.
     *
     * <p>{@code clear()} empties the box without the keystrokes this page's
     * controlled inputs listen for, which leaves the component holding text that
     * is no longer on screen. Backspacing over the value cannot be missed.
     */
    private void fill(String id, String value) {
        WebElement field = find(id);
        field.click();
        field.sendKeys(Keys.END);
        String current = field.getDomProperty("value");
        if (!current.isEmpty()) {
            field.sendKeys(String.valueOf(Keys.BACK_SPACE).repeat(current.length()));
        }
        if (!value.isEmpty()) {
            field.sendKeys(value);
        }
    }

    /** Waits until this many elements carry the test id: every change here is a fetch. */
    private void waitForCount(String id, int expected) {
        wait.until((ExpectedCondition<Boolean>) d -> d.findElements(testId(id)).size() == expected);
    }

    private void waitForAttribute(String id, String attribute, String expected) {
        wait.until((ExpectedCondition<Boolean>) d -> {
            List<WebElement> found = d.findElements(testId(id));
            return !found.isEmpty() && expected.equals(found.get(0).getAttribute(attribute));
        });
    }

    private void waitForTextContaining(String id, String fragment) {
        wait.until((ExpectedCondition<Boolean>) d -> {
            List<WebElement> found = d.findElements(testId(id));
            return !found.isEmpty() && found.get(0).getText().contains(fragment);
        });
    }
}
