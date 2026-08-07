package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.List;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.support.ui.ExpectedCondition;

/** /app/autosave — an indicator that says "saved" before anything was saved, and the version that is the only fact worth waiting for. */
class AutosaveTest extends Playground {

    private static final String PAGE = "/app/autosave";
    private static final String RECORD = "/api/app/autosave/record?latencyMs=0";
    private static final String OTHER_WRITER = "/api/app/autosave/other-writer";

    /**
     * How long typing must stop for, in every test here that wants exactly one
     * write.
     *
     * <p>Wider than the challenge's own default suggests, and the reason is a
     * real difference between the two suites. Playwright's {@code fill} puts a
     * field into its final state in one operation and fires one input event;
     * {@code sendKeys} fires one per character, and every one of them restarts
     * the debounce. Under the hundred milliseconds the Playwright spec uses,
     * WebDriver would send a write halfway through the word and a second one
     * after it -- two versions where the test expected one. Widening the window
     * the challenge already exposes as a query parameter is the honest fix;
     * loosening the assertion to "some version above one" would throw away the
     * thing being taught.
     */
    private static final int DEBOUNCE_MS = 600;

    /** Long enough that "saving" is a state a poll can see, and that the guard has something to guard. */
    private static final int SLOW_WRITE_MS = 2000;

    @Test
    void theRecordStartsUnwrittenAndTheIndicatorAlreadySaysSaved() {
        open(PAGE);

        waitForText("record-version", "1");
        assertEquals("nobody", text("updated-by"));
        assertEquals("0", text("save-count"));
        assertEquals("saved", text("save-state"));
    }

    /**
     * The trap, and the whole point of the page. The indicator describes the
     * autosave loop, not the draft, so it is still showing the word it was
     * showing before anything was typed -- and a test that waits for it goes
     * green having saved nothing.
     */
    @Test
    void waitingForSavedIsSatisfiedByTheWordThatWasAlreadyThere() {
        open(PAGE + "?debounceMs=10000");
        waitForText("record-version", "1");

        fill("field-title", "Winter timetable");

        // This passes, and it passes instantly, with ten seconds of debounce
        // still ahead of it.
        waitForText("save-state", "saved");

        // What actually happened: nothing left the browser.
        assertEquals("1", text("record-version"));
        assertEquals("0", text("save-count"));
    }

    @Test
    void theVersionIsTheFactWorthWaitingFor() {
        open(PAGE + "?debounceMs=" + DEBOUNCE_MS + "&latencyMs=100");
        waitForText("record-version", "1");

        fill("field-title", "Winter timetable");

        // Only the server moves either of these, so neither can be satisfied by
        // a value the page was already showing.
        waitForText("record-version", "2");
        waitForText("save-count", "1");
        assertEquals("this page", text("updated-by"));

        assertEquals("Winter timetable", api("GET", RECORD, null, "b.record.fields.title"));
    }

    @Test
    void theMiddleStateIsRealAndItIsWhereTheGuardApplies() {
        open(PAGE + "?debounceMs=" + DEBOUNCE_MS + "&latencyMs=" + SLOW_WRITE_MS);
        waitForText("record-version", "1");

        fill("field-owner", "Dana Okonkwo");
        waitForText("save-state", "saving");
        waitForText("save-state", "saved");
        waitForText("save-count", "1");
    }

    @Test
    void theServerRefusesAStaleWriteAndHandsBackWhatItIsHolding() {
        // No page needed: the base class has already parked the browser on the
        // playground's own origin with this test's session cookie set, so a
        // fetch from here lands in this test's record like any other request.
        String fields = "\"fields\":{\"title\":\"Once\",\"owner\":\"unassigned\",\"notes\":\"n\"}";

        assertEquals(
                "200 v2",
                api("PUT", RECORD, "{\"version\":1," + fields + "}", "r.status + ' v' + b.record.version"));

        assertEquals(
                "409 v2 Once",
                api("PUT", RECORD,
                        "{\"version\":1,\"fields\":{\"title\":\"Twice\",\"owner\":\"unassigned\",\"notes\":\"n\"}}",
                        "r.status + ' v' + b.record.version + ' ' + b.record.fields.title"),
                "a refused write must not half-apply");
    }

    @Test
    void theOtherWriterCanBeAimedAndRefusesAFieldThatDoesNotExist() {
        assertEquals(
                "200 Cancelled in high winds.",
                api("POST", OTHER_WRITER, "{\"field\":\"notes\",\"value\":\"Cancelled in high winds.\"}",
                        "r.status + ' ' + b.record.fields.notes"));

        // Refused rather than ignored: a bumped version for a misspelt field
        // would send you looking for the bug in the page.
        String wrong = api("POST", OTHER_WRITER, "{\"field\":\"titel\",\"value\":\"x\"}", "r.status + ' ' + b.error");
        assertTrue(wrong.startsWith("400"), wrong);
        assertTrue(wrong.contains("no such field"), wrong);
    }

    @Test
    void theButtonPlaysTheOtherWriterAndTellsThisPageNothing() {
        open(PAGE);
        waitForText("other-writer-note", "nobody else has written yet");

        click("simulate-other-writer");
        waitForTextContaining("other-writer-note", "version 2");

        // The page is still holding the version it loaded, which is the position
        // a real editor would be in and the reason its next autosave collides.
        assertEquals("1", text("record-version"));
    }

    @Test
    void aStaleAutosaveSurfacesAsAConflictRatherThanAsASave() {
        conflict();

        assertTrue(text("conflict-versions").contains("version 1"));
        assertTrue(text("conflict-versions").contains("version 2"));

        // The field still shows text the server has never accepted, and the word
        // the indicator offers for that is "idle".
        assertEquals("Winter timetable", value("field-title"));
        assertEquals("idle", text("save-state"));
        assertEquals("0", text("save-count"));
    }

    @Test
    void keepingYoursDiscardsTheChangeYouNeverSaw() {
        conflict();

        click("keep-mine");
        waitForAbsent("conflict");
        waitForText("record-version", "3");
        waitForText("save-count", "1");

        assertEquals("Winter timetable", api("GET", RECORD, null, "b.record.fields.title"));
        assertEquals(
                "unassigned",
                api("GET", RECORD, null, "b.record.fields.owner"),
                "their edit is gone and nothing said so");
    }

    @Test
    void takingTheirsDiscardsYoursAndWritesNothingAtAll() {
        conflict();

        click("take-theirs");
        waitForAbsent("conflict");

        assertEquals("Ferry timetable rewrite", value("field-title"));
        assertEquals("Priya Raman", value("field-owner"));
        assertEquals("2", text("record-version"));
        assertEquals("another writer", text("updated-by"));

        // Adopting a record is not a write, so nothing was acknowledged.
        assertEquals("0", text("save-count"));
    }

    @Test
    void onlyTheMergeKeepsBothChanges() {
        conflict();

        click("show-merge");
        waitForPresent("merge-view");

        // The merge opens on the union of the two sets of edits: the field this
        // page changed takes mine, the field only they changed takes theirs.
        waitForCell("title", "merge-choice", "mine");
        waitForCell("owner", "merge-choice", "theirs");
        waitForCell("title", "merge-theirs", "Ferry timetable rewrite");
        waitForCell("owner", "merge-mine", "unassigned");

        cell("owner", "merge-pick").click();
        waitForCell("owner", "merge-choice", "mine");
        cell("owner", "merge-pick").click();
        waitForCell("owner", "merge-choice", "theirs");

        click("save-merge");
        waitForAbsent("conflict");
        waitForText("record-version", "3");

        assertEquals("Winter timetable", api("GET", RECORD, null, "b.record.fields.title"));
        assertEquals("Priya Raman", api("GET", RECORD, null, "b.record.fields.owner"));
    }

    @Test
    void leavingIsRefusedWhileAWriteIsInFlight() {
        open(PAGE + "?debounceMs=" + DEBOUNCE_MS + "&latencyMs=" + SLOW_WRITE_MS);
        waitForText("record-version", "1");

        fill("field-notes", "Four crossings in summer, weather permitting.");
        waitForText("save-state", "saving");

        click("leave-link");
        waitForPresent("leave-blocked");
        waitForPresent("field-notes");

        // The same click, once the server has acknowledged, goes through.
        waitForText("save-count", "1");
        click("leave-link");
        waitForAbsent("field-notes");
    }

    @Test
    void theBeforeUnloadGuardExistsOnlyWhileAWriteIsInFlight() {
        open(PAGE + "?debounceMs=" + DEBOUNCE_MS + "&latencyMs=" + SLOW_WRITE_MS);
        waitForText("record-version", "1");

        assertFalse(guarded(), "a page with nothing outstanding must not nag on the way out");

        fill("field-title", "Winter timetable");
        waitForText("save-state", "saving");
        assertTrue(guarded(), "an unacknowledged edit was not protected");

        waitForText("save-count", "1");
        assertFalse(guarded(), "the guard outlived the write it was guarding");
    }

    // --- the conflict every resolution test starts from ----------------------

    /**
     * Loads the page, lets a second writer move the record while this page is
     * not looking, and then edits a different field so the next autosave
     * collides.
     *
     * <p>Waiting for version one first is what makes the collision certain: an
     * other-writer call that landed before the page had read the record would be
     * the version the page loaded, and there would be nothing to conflict with.
     */
    private void conflict() {
        open(PAGE + "?debounceMs=" + DEBOUNCE_MS + "&latencyMs=100");
        waitForText("record-version", "1");

        api("POST", OTHER_WRITER, null, "r.status");

        fill("field-title", "Winter timetable");
        waitForPresent("conflict");
    }

    // --- helpers the base class does not carry -------------------------------

    /** Replaces a box's contents, because sendKeys alone appends to what is already there. */
    private void fill(String id, String value) {
        WebElement box = find(id);
        box.clear();
        box.sendKeys(value);
    }

    /** What a box currently holds. The DOM property, not the attribute: the attribute keeps the value the page was built with. */
    private String value(String id) {
        return find(id).getDomProperty("value");
    }

    private void waitForTextContaining(String id, String fragment) {
        wait.until((ExpectedCondition<Boolean>) d -> {
            List<WebElement> found = d.findElements(testId(id));
            return !found.isEmpty() && found.get(0).getText().contains(fragment);
        });
    }

    /** One cell of one merge row: the row is narrowed by its field, the cell by its test id within it. */
    private WebElement cell(String field, String id) {
        return wait.until(d -> {
            List<WebElement> rows = d.findElements(
                    By.cssSelector("[data-testid='merge-row'][data-field='" + field + "']"));
            if (rows.isEmpty()) {
                return null;
            }
            List<WebElement> cells = rows.get(0).findElements(testId(id));
            return cells.isEmpty() ? null : cells.get(0);
        });
    }

    private void waitForCell(String field, String id, String expected) {
        wait.until((ExpectedCondition<Boolean>) d -> cell(field, id).getText().trim().equals(expected));
    }

    /**
     * Whether the page is currently refusing to be unloaded.
     *
     * <p>Dispatched rather than navigated, exactly as the Playwright spec does
     * it: whether the browser really shows the prompt depends on interaction
     * heuristics that differ per engine, and the handler's presence is the
     * contract. WebDriver's own dialog handling would not help here either -- it
     * dismisses beforeunload prompts before a test can see them.
     */
    private boolean guarded() {
        return (Boolean) ((JavascriptExecutor) driver).executeScript(
                "const event = new Event('beforeunload', { cancelable: true });"
                        + "window.dispatchEvent(event);"
                        + "return event.defaultPrevented;");
    }

    /**
     * Sends a request the page would not send, and hands back one value from the
     * answer as a string.
     *
     * <p>The Playwright suite has an HTTP client beside the page for this.
     * WebDriver has none, so the fetch is driven from page script -- the same
     * request from a different place: the session cookie is already set, so it
     * lands in this test's record, and unlike navigating to the endpoint it
     * leaves the page standing where it is. The expression is evaluated against
     * the response {@code r} and its parsed body {@code b}, so status and body
     * can be read from one round trip rather than two, which for a versioned
     * record matters: a second request would be asking about a different
     * version.
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
