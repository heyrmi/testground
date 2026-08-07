package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertNotEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.time.Duration;
import java.util.List;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.support.ui.ExpectedCondition;
import org.openqa.selenium.support.ui.Select;
import org.openqa.selenium.support.ui.WebDriverWait;

/** /app/admin-crud — writes that land on the page before the server agrees, a delete that has not been sent yet, and a selection that outlives the filter that made it. */
class AdminCrudTest extends Playground {

    private static final String PAGE = "/app/admin-crud";
    private static final String ACCOUNTS = "/api/app/admin-crud/accounts";

    private static final By ROWS = testId("account-row");
    /** The seeded accounts the server refuses to change or delete, published on the row so a test can choose the rollback path. */
    private static final By LOCKED_ROWS = By.cssSelector("[data-testid='account-row'][data-locked='true']");
    /** A row the page has drawn under an id the server has not issued. */
    private static final By OPTIMISTIC_ROWS = By.cssSelector("[data-testid='account-row'][data-id^='tmp-']");

    @Test
    void theTableStartsAsTwelveAccountsThreeOfWhichTheServerWillNotChange() {
        open(PAGE);

        waitForCount(ROWS, 12);
        assertEquals("12 of 12", text("row-count"));
        assertEquals(3, driver.findElements(LOCKED_ROWS).size());
        assertEquals("0", text("queued-deletes"));
        assertEquals("0", text("in-flight"));
    }

    /**
     * The id is the thing that changes, so it is the one attribute a locator must
     * not be built from before the server has answered.
     */
    @Test
    void aCreatedRowIsNotTheRowTheServerStored() {
        open(PAGE + "?latencyMs=800");
        waitForCount(ROWS, 12);

        fill("new-name", "Wilhelmina Vandertramp");
        click("create-account");

        waitForCount(OPTIMISTIC_ROWS, 1);
        assertEquals("creating", driver.findElement(OPTIMISTIC_ROWS).findElement(testId("account-state")).getText());

        // Settling is what makes the next assertion mean anything.
        waitForText("in-flight", "0");

        assertEquals(0, driver.findElements(OPTIMISTIC_ROWS).size());
        waitForRowText("acct-13", "account-name", "Wilhelmina Vandertramp");
        waitForRowText("acct-13", "account-state", "saved");
    }

    @Test
    void aCreateTheServerRefusesLeavesARowThatWasNeverThere() {
        open(PAGE + "?latencyMs=800");
        waitForCount(ROWS, 12);

        String taken = rowText("acct-1", "account-name");
        fill("new-name", taken);
        click("create-account");

        // Thirteen rows, one of which the server is about to refuse to store.
        waitForCount(ROWS, 13);
        waitForCount(ROWS, 12);

        assertTrue(text("rollback-notice").contains("already exists"));
        assertEquals("1", text("rollback-count"));
    }

    @Test
    void anEditTheServerRefusesIsUndoneAfterItHasAlreadyBeenShown() {
        open(PAGE + "?latencyMs=1000");
        waitForCount(ROWS, 12);

        String locked = firstLockedId();
        String before = rowText(locked, "account-name");

        clickInRow(locked, "row-edit");
        fill("edit-name", "Renamed by the suite");
        click("edit-save");

        waitForRowText(locked, "account-name", "Renamed by the suite");
        waitForRowText(locked, "account-state", "saving");

        waitForText("in-flight", "0");
        waitForRowText(locked, "account-name", before);
        assertTrue(text("rollback-notice").contains("locked"));
    }

    /**
     * The approach that looks like it works. It will keep passing while the write
     * silently fails, because it asserts on a value the client invented.
     */
    @Test
    void assertingStraightAfterTheClickPassesAgainstAStateTheServerNeverHad() {
        open(PAGE + "?latencyMs=1500");
        waitForCount(ROWS, 12);

        String locked = firstLockedId();

        clickInRow(locked, "row-edit");
        fill("edit-name", "Never stored");
        click("edit-save");

        waitForRowText(locked, "account-name", "Never stored");

        assertNotEquals(
                "Never stored",
                api("GET", ACCOUNTS, null, "b.accounts.find(a => a.id === '" + locked + "').name"),
                "the page said one thing and the server holds another");
    }

    @Test
    void aQueuedDeleteHasNotBeenSentAndUndoMeansTheServerNeverHearsAboutIt() {
        open(PAGE + "?latencyMs=100&undoMs=8000");
        waitForCount(ROWS, 12);

        clickInRow("acct-1", "row-delete");
        waitForCount(rowBy("acct-1"), 0);
        waitForPresent("undo-toast");
        assertEquals("1", text("queued-deletes"));

        // Gone from the page, untouched on the server. Reading it here is reading
        // too early rather than reading a stale value.
        assertEquals("true", api("GET", ACCOUNTS, null, "b.accounts.some(a => a.id === 'acct-1')"));

        click("undo-delete");
        waitForCount(rowBy("acct-1"), 1);
        waitForAbsent("undo-toast");
        assertEquals("0", text("in-flight"));
    }

    @Test
    void theDeleteLeavesWhenTheWindowClosesAndALockedAccountComesBack() {
        open(PAGE + "?latencyMs=400&undoMs=400");
        waitForCount(ROWS, 12);

        String locked = firstLockedId();
        clickInRow(locked, "row-delete");
        waitForCount(rowBy(locked), 0);

        // Two things settle here, and only in this order: the window closes, then
        // the request the window was holding is answered. Waiting on one of them
        // and calling it settled is the mistake this pair of counters exists to
        // make visible.
        waitForText("queued-deletes", "0");
        waitForText("in-flight", "0");

        waitForCount(rowBy(locked), 1);
        assertTrue(text("rollback-notice").contains("locked"));
        assertEquals("1", text("rollback-count"));
    }

    @Test
    void selectAllCoversTheRowsTheFilterLeftNotTheTable() {
        open(PAGE);
        waitForCount(ROWS, 12);

        new Select(find("role-filter")).selectByValue("editor");
        waitForCount(ROWS, 4);

        click("select-all");

        // Four, not twelve. In an admin UI the difference between those two
        // numbers is the whole of the damage.
        assertEquals("4", text("selected-count"));
        assertEquals("4 of 12", text("row-count"));
    }

    @Test
    void theSelectionOutlivesTheFilterThatMadeIt() {
        open(PAGE + "?latencyMs=800&undoMs=0");
        waitForCount(ROWS, 12);

        new Select(find("role-filter")).selectByValue("viewer");
        waitForText("row-count", "4 of 12");
        click("select-all");
        assertEquals("4", text("selected-count"));

        new Select(find("role-filter")).selectByValue("");
        waitForCount(ROWS, 12);

        // The box reads unchecked because not every visible row is chosen. The
        // four it did choose are still selected, and still about to be deleted.
        assertFalse(find("select-all").isSelected());
        assertEquals("4", text("selected-count"));

        click("bulk-delete");
        waitForCount(ROWS, 8);

        // Three deletes are kept and the locked one is undone, so the count that
        // matters is the one after everything settles rather than the one on
        // screen.
        waitForCount(ROWS, 9);
        assertEquals("1", text("rollback-count"));
        assertEquals("0", text("in-flight"));

        assertEquals("9", api("GET", ACCOUNTS, null, "b.accounts.length"));
    }

    // --- helpers the base class does not carry -------------------------------

    /**
     * A wait that polls twenty times a second rather than twice.
     *
     * <p>Several of the states this page is about exist for a few hundred
     * milliseconds: the optimistic row before the server answers, the thirteenth
     * row that is about to be taken away, the eight rows before the locked one
     * comes back. Playwright's assertions poll far faster than
     * {@code WebDriverWait}'s half-second default, so a straight translation
     * would step over the very state the challenge exists to show and fail with
     * the count that comes after it. Polling faster is the fix; widening the
     * challenge's latency until the default poll can see it would be testing a
     * page nobody ships.
     */
    private WebDriverWait quickly() {
        return new WebDriverWait(driver, TIMEOUT, Duration.ofMillis(50));
    }

    private void waitForCount(By by, int expected) {
        quickly().until((ExpectedCondition<Boolean>) d -> d.findElements(by).size() == expected);
    }

    private static By rowBy(String id) {
        return By.cssSelector("[data-testid='account-row'][data-id='" + id + "']");
    }

    /** One cell of one row. Re-found on every call, because a restored row is a new node and a captured one would be stale. */
    private WebElement inRow(String rowId, String childId) {
        return quickly().until(d -> {
            List<WebElement> rows = d.findElements(rowBy(rowId));
            if (rows.isEmpty()) {
                return null;
            }
            List<WebElement> cells = rows.get(0).findElements(testId(childId));
            return cells.isEmpty() ? null : cells.get(0);
        });
    }

    private String rowText(String rowId, String childId) {
        return inRow(rowId, childId).getText().trim();
    }

    private void waitForRowText(String rowId, String childId, String expected) {
        quickly().until((ExpectedCondition<Boolean>) d -> rowText(rowId, childId).equals(expected));
    }

    private void clickInRow(String rowId, String childId) {
        inRow(rowId, childId).click();
    }

    /** The id of the first account the server refuses to touch. Published on the row, so this is a choice rather than a discovery. */
    private String firstLockedId() {
        return wait.until(d -> {
            List<WebElement> locked = d.findElements(LOCKED_ROWS);
            return locked.isEmpty() ? null : locked.get(0).getDomAttribute("data-id");
        });
    }

    /** Replaces a box's contents, because sendKeys alone appends to what is already there. */
    private void fill(String id, String value) {
        WebElement box = find(id);
        box.clear();
        box.sendKeys(value);
    }

    /**
     * Asks the server what it actually holds, and hands back one value as a
     * string.
     *
     * <p>The Playwright suite has an HTTP client beside the page for this.
     * WebDriver has none, so the fetch is driven from page script -- the same
     * request from a different place: the session cookie is already set, so it
     * reads this test's accounts, and unlike navigating to the endpoint it
     * leaves the table standing where it is. That matters more here than
     * anywhere else in this suite: a queued delete lives in the page, so
     * navigating away to read the server would cancel the very request the
     * assertion is about.
     */
    private String api(String method, String path, String body, String expression) {
        driver.manage().timeouts().scriptTimeout(TIMEOUT);
        return (String) ((JavascriptExecutor) driver).executeAsyncScript(
                "const [method, path, body] = arguments;"
                        + "const done = arguments[arguments.length - 1];"
                        + "const init = body === null"
                        + "  ? { method }"
                        + "  : { method, headers: { 'Content-Type': 'application/json' }, body };"
                        + "fetch(path, init)"
                        + "  .then(async r => { const b = await r.json(); done(String(" + expression + ")); })"
                        + "  .catch(e => done('the request failed: ' + e));",
                method, path, body);
    }
}
