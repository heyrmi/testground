package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.List;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.support.ui.ExpectedCondition;
import org.openqa.selenium.support.ui.Select;

/** /app/wizard — validation that runs on advance, a branch chosen three steps back, and a server told about a step only when Next clears it. */
class WizardTest extends Playground {

    private static final String PAGE = "/app/wizard";

    @Test
    void fourStepsAReviewAndAReference() {
        open(PAGE);
        waitForContent("step-counter", "Step 1 of 4");

        account("individual", "sam@example.test");
        contact();
        individualDetails("1990-04-12");

        waitForContent("step-counter", "Step 4 of 4");
        assertEquals("sam@example.test", reviewValue("email"));

        click("submit");
        assertTrue(text("reference").matches("WZ-\\d+"), "the confirmation did not carry a reference");
    }

    /**
     * The trap. Waiting for Next to become enabled is waiting for something that
     * was already true, and the error worth asserting on does not exist until
     * after the click -- so a test written the obvious way passes on a page that
     * refused every box.
     */
    @Test
    void nextIsEnabledOnAnInvalidStepAndTheErrorOnlyExistsAfterTheClick() {
        open(PAGE);

        assertTrue(find("next").isEnabled());
        assertEquals(0, count("field-error"));

        click("next");

        // Still on step one, which is the only thing the click's own outcome tells us.
        waitForContent("step-counter", "Step 1 of 4");
        assertTrue(fieldError("account-type").contains("individual or a business"));
        assertTrue(fieldError("email").contains("not an email address"));
    }

    @Test
    void stepThreeAsksABusinessDifferentQuestions() {
        open(PAGE);
        account("business", "sam@example.test");
        contact();

        waitForContent("branch", "business");
        waitForPresent("company-number");

        // Branch on the answer, not on the step number. A locator hard-coded to
        // the other branch reports a missing element, which reads as a broken
        // selector rather than as an answer two steps back deciding what exists.
        assertEquals(0, count("date-of-birth"));

        fill("company-number", "01234567");
        fill("employees", "12");
        click("next");

        click("submit");
        waitForPresent("reference");
    }

    @Test
    void goingBackKeepsWhatWasTypedAndTellsTheServerNothing() {
        open(PAGE);
        account("individual", "sam@example.test");

        fill("full-name", "Sam Okafor");
        click("back");
        assertEquals("sam@example.test", value("email"));

        stepLink(2).click();
        assertEquals("Sam Okafor", value("full-name"));

        assertNull(draft("d.values['full-name']"), "a step is only stored when Next validates it");
        assertEquals("1", draft("d.steps.join(',')"));
    }

    /** The cause is on step one and the message arrives on step four. */
    @Test
    void anAddressStepOneAcceptedIsRefusedAtTheFarEndOfTheFlow() {
        open(PAGE);
        account("individual", "sam@rejected.test");

        // Step one took it: the page checks the shape and nothing else.
        waitForContent("step-counter", "Step 2 of 4");

        contact();
        individualDetails("1990-04-12");
        click("submit");

        WebElement refusal = narrow("problem", "email");
        assertTrue(refusal.getText().contains("rejected.test"));
        assertEquals("1", refusal.getDomAttribute("data-step"), "the refusal did not name the step that caused it");
        assertEquals(0, count("reference"));
    }

    @Test
    void thePageChecksADateOfBirthForItsShapeAndTheServerForItsMeaning() {
        // The age rule is measured against the session clock, so "too young"
        // would otherwise mean something different every year this suite runs.
        pinClock();

        open(PAGE);
        account("individual", "sam@example.test");
        contact();

        fill("date-of-birth", "12 April 1990");
        fill("occupation", "Ceramicist");
        click("next");
        assertTrue(fieldError("date-of-birth").contains("YYYY-MM-DD"));

        // Well formed, so the page waves it through; too young, so the server does not.
        fill("date-of-birth", "2020-04-12");
        click("next");
        waitForContent("step-counter", "Step 4 of 4");

        click("submit");
        assertTrue(narrow("problem", "date-of-birth").getText().contains("eighteen"));
    }

    /**
     * The looks-like-it-works case. Everything on screen says business, the
     * application that was lodged says individual, and nothing failed.
     */
    @Test
    void anAnswerChangedAndJumpedPastNeverReachesTheServer() {
        open(PAGE);
        account("individual", "sam@example.test");
        contact();
        individualDetails("1990-04-12");

        stepLink(1).click();
        new Select(find("account-type")).selectByValue("business");
        stepLink(4).click();

        waitForContent("branch", "business");
        assertEquals("not answered", reviewValue("company-number"));

        click("submit");
        waitForPresent("reference");

        assertEquals(
                "individual",
                draft("d.applications[0].values['account-type']"),
                "the review showed one application and the server processed another");
    }

    @Test
    void aStepSkippedAfterTheBranchChangedIsRefusedThreeStepsLater() {
        open(PAGE);
        account("individual", "sam@example.test");
        contact();
        individualDetails("1990-04-12");

        stepLink(1).click();
        new Select(find("account-type")).selectByValue("business");
        click("next"); // this one the server does hear
        stepLink(4).click(); // step three is never revisited, so nothing re-checks it

        click("submit");

        assertTrue(text("submit-error").contains("does not validate"));
        assertEquals("3", narrow("problem", "company-number").getDomAttribute("data-step"));
        assertTrue(narrow("problem", "employees").isDisplayed());
    }

    @Test
    void theStepIsNotInTheUrlSoAReloadRestartsAFlowTheServerStillRemembers() {
        open(PAGE);
        account("individual", "sam@example.test");
        contact();

        waitForContent("step-counter", "Step 3 of 4");
        assertTrue(driver.getCurrentUrl().endsWith("/app/wizard"), "the step leaked into the address bar");

        driver.navigate().refresh();

        waitForContent("step-counter", "Step 1 of 4");
        assertEquals("", value("email"));

        assertEquals("sam@example.test", draft("d.values.email"), "the page forgot; the server did not");
        assertEquals("1,2", draft("d.steps.join(',')"));
    }

    @Test
    void theServerValidatesTheDraftItHoldsNotTheRequestBody() {
        open(PAGE);

        String refused = post(
                "/api/app/wizard/submit",
                "{\"values\":{\"account-type\":\"individual\",\"email\":\"sam@example.test\"}}");

        assertTrue(refused.startsWith("409"), "a submit with a body but no walked flow was not refused: " + refused);
        assertTrue(refused.contains("no draft to submit"));
    }

    // --- the flow, in the three shapes every test above needs ---------------

    private void account(String type, String email) {
        new Select(find("account-type")).selectByValue(type);
        fill("email", email);
        click("next");
    }

    private void contact() {
        fill("full-name", "Sam Okafor");
        fill("phone", "020 7946 0018");
        click("next");
    }

    private void individualDetails(String born) {
        fill("date-of-birth", born);
        fill("occupation", "Ceramicist");
        click("next");
    }

    // --- helpers the base class does not carry ------------------------------

    /**
     * Waits for the text the DOM holds, which is not the text the page paints.
     *
     * <p>A real difference between the two suites rather than a preference.
     * Playwright's {@code toHaveText} reads {@code textContent}; WebDriver's
     * {@code getText} returns *rendered* text, and the step counter sits in the
     * page's label style, which uppercases. So {@code getText} says
     * "STEP 1 OF 4" for markup that says "Step 1 of 4", and a test that took
     * the base class's {@code waitForText} would fail on a stylesheet rather
     * than on the wizard. Reading the property keeps the assertion about the
     * step number.
     */
    private void waitForContent(String id, String expected) {
        wait.until((ExpectedCondition<Boolean>) d -> {
            List<WebElement> found = d.findElements(testId(id));
            return !found.isEmpty() && expected.equals(found.get(0).getDomProperty("textContent").trim());
        });
    }

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

    /**
     * The one element carrying this test id and this {@code data-field}.
     *
     * <p>Playwright narrows a located set with a chained locator; the nearest
     * thing here is to put both attributes in one selector. It is still the
     * published contract being used -- {@code data-field} is declared alongside
     * the test id -- but it has to be spelled out rather than composed.
     */
    private WebElement narrow(String id, String field) {
        By by = By.cssSelector("[data-testid='" + id + "'][data-field='" + field + "']");
        return wait.until(d -> {
            List<WebElement> found = d.findElements(by);
            return found.isEmpty() ? null : found.get(0);
        });
    }

    private String fieldError(String field) {
        return narrow("field-error", field).getText().trim();
    }

    private String reviewValue(String field) {
        return narrow("review-value", field).getText().trim();
    }

    private WebElement stepLink(int step) {
        By by = By.cssSelector("[data-testid='step-link'][data-step='" + step + "']");
        return wait.until(d -> {
            List<WebElement> found = d.findElements(by);
            return found.isEmpty() ? null : found.get(0);
        });
    }

    /**
     * Asks the server what it actually holds, and hands one value back.
     *
     * <p>The Playwright suite has an HTTP client beside the page for this.
     * WebDriver has none, so the fetch is driven from page script -- which is
     * not a workaround so much as the same request from a different place: the
     * session cookie is already set, so it lands in this test's draft, and
     * unlike navigating to the endpoint it leaves the wizard standing where it
     * is. A test that navigated away to read the draft would have destroyed the
     * step state it was about to assert on.
     */
    private Object draft(String expression) {
        driver.manage().timeouts().scriptTimeout(TIMEOUT);
        return ((JavascriptExecutor) driver).executeAsyncScript(
                "const done = arguments[arguments.length - 1];"
                        + "fetch('/api/app/wizard/draft').then(r => r.json())"
                        + "  .then(d => done(" + expression + " ?? null))"
                        + "  .catch(e => done('the draft could not be read: ' + e));");
    }

    /** Posts a body the page would never send, and reports the status and the refusal together. */
    private String post(String path, String body) {
        driver.manage().timeouts().scriptTimeout(TIMEOUT);
        return (String) ((JavascriptExecutor) driver).executeAsyncScript(
                "const done = arguments[arguments.length - 1];"
                        + "fetch(arguments[0], {"
                        + "  method: 'POST',"
                        + "  headers: { 'Content-Type': 'application/json' },"
                        + "  body: arguments[1],"
                        + "}).then(async r => done(r.status + ' ' + ((await r.json()).error ?? '')))"
                        + "  .catch(e => done('0 ' + e));",
                path, body);
    }

    private void pinClock() {
        assertTrue(
                post("/api/control/clock", "{\"action\":\"set\",\"instant\":\"2026-01-01T00:00:00Z\"}").startsWith("200"),
                "the session clock could not be pinned, so an age assertion would drift with today's date");
    }
}
