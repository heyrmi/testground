package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.Keys;
import org.openqa.selenium.WebElement;

/** /app/otp-input — six one-character boxes that move the focus out from under you. */
class OtpInputTest extends Playground {

    private static final String PAGE = "/app/otp-input";
    private static final String CODE = "314159";

    @Test
    void typingTheWholeCodeAtTheFirstBoxWorksBecauseTheKeysFollowTheFocus() {
        open(PAGE);

        // This is where the two reference suites genuinely part company, and the
        // difference is worth stating rather than papering over. Playwright's
        // fill() is a bulk write of the value, which the box's maxlength cuts to
        // one character, so the companion spec asserts that five digits are
        // silently dropped. sendKeys is real typing: each key is delivered to
        // whatever holds the focus at that instant, and the page moves the focus
        // along after every digit, so the identical-looking call fills all six
        // boxes. Neither tool is wrong; they model different user actions.
        find("otp-0").sendKeys(CODE);

        waitForText("otp-value", CODE);
        waitForText("otp-verdict", "accepted");
    }

    @Test
    void oneDigitPerBoxIsTheApproachThatDoesNotDependOnFocusChasing() {
        open(PAGE);

        // Addressing each box by its own test id is the version that keeps
        // working if the page ever stops advancing the focus for us.
        for (int index = 0; index < CODE.length(); index++) {
            find("otp-" + index).sendKeys(String.valueOf(CODE.charAt(index)));
        }

        waitForText("otp-value", CODE);
        waitForText("otp-verdict", "accepted");
    }

    @Test
    void typingAdvancesTheFocusWithoutBeingToldTo() {
        open(PAGE);

        find("otp-0").sendKeys("3");
        assertEquals("otp-1", focusedTestId(), "the page should have moved the focus on for us");

        // Sending to the driver's active element rather than to a located one
        // proves the focus really moved, instead of us putting it back.
        driver.switchTo().activeElement().sendKeys("1");
        assertEquals("otp-2", focusedTestId());
    }

    @Test
    void backspaceOnAnEmptyBoxWalksBackwards() {
        open(PAGE);
        find("otp-0").sendKeys("3");
        find("otp-1").sendKeys("1");

        find("otp-2").sendKeys(Keys.BACK_SPACE);

        assertEquals("otp-1", focusedTestId(), "backspace in an empty box should retreat to the last filled one");
    }

    @Test
    void pastingSpreadsTheCodeAcrossEveryBoxInOneStep() {
        open(PAGE);
        WebElement first = find("otp-0");
        first.click();

        // WebDriver has no portable way to put text on the system clipboard, and
        // a real Ctrl+V would read whatever the machine happened to be holding.
        // Dispatching the paste event with its own DataTransfer is the honest
        // route -- the page's onPaste handler is what is under test, not the OS.
        ((JavascriptExecutor) driver).executeScript(
                "const transfer = new DataTransfer();"
                        + "transfer.setData('text', arguments[1]);"
                        + "arguments[0].dispatchEvent(new ClipboardEvent('paste',"
                        + "  { clipboardData: transfer, bubbles: true, cancelable: true }));",
                first, CODE);

        waitForText("otp-value", CODE);
        waitForText("otp-verdict", "accepted");
    }

    @Test
    void aWrongCodeIsRejectedRatherThanLeftIncomplete() {
        open(PAGE);

        for (int index = 0; index < 6; index++) {
            find("otp-" + index).sendKeys("9");
        }

        // Six digits and still refused: incomplete and rejected are different
        // verdicts, and a test that only checked for "not accepted" would not
        // notice a page that had stopped assembling the code at all.
        waitForText("otp-verdict", "rejected");
    }

    @Test
    void settingTheValueWithScriptFillsTheBoxAndTellsThePageNothing() {
        open(PAGE);
        WebElement first = find("otp-0");

        // The trap. Assigning .value is the fast way past a fiddly input, and it
        // even survives maxlength, so a screenshot of the failure shows a box
        // holding the whole code. React never hears about it: the state behind
        // the controlled input is untouched, so the assembled code is still
        // empty and the next render would wipe the box.
        ((JavascriptExecutor) driver).executeScript("arguments[0].value = arguments[1];", first, CODE);

        assertEquals(CODE, first.getDomProperty("value"), "the DOM really was written to");
        assertEquals("(empty)", text("otp-value"), "the page must not have seen a script assignment");
        assertEquals("nothing entered", text("otp-verdict"));
    }

    @Test
    void clearingPutsEveryBoxBack() {
        open(PAGE);
        find("otp-0").sendKeys(CODE);
        waitForText("otp-verdict", "accepted");

        click("otp-clear");

        waitForText("otp-value", "(empty)");
        waitForText("otp-verdict", "incomplete");
    }

    /** The test id of whatever currently holds the focus, or null if it is the body. */
    private String focusedTestId() {
        return (String) ((JavascriptExecutor) driver)
                .executeScript("return document.activeElement.getAttribute('data-testid');");
    }
}
