package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.Alert;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.UnhandledAlertException;
import org.openqa.selenium.support.ui.ExpectedConditions;

/**
 * /legacy/native-dialogs — alert, confirm and prompt are browser state, not DOM,
 * and WebDriver reaches them through a door of their own.
 */
class NativeDialogsTest extends Playground {

    private static final String PAGE = "/legacy/native-dialogs";

    @Test
    void theOnlyPlaceTheMessageExistsIsTheAlertItself() {
        open(PAGE);
        click("fire-alert");

        // The two frameworks invert each other here and both are right.
        // Playwright registers a handler before the click, because the dialog
        // blocks the page and there is no "after" until it is answered.
        // WebDriver has no handler to register: the dialog is a piece of
        // browser state you switch to once it exists, so the order is reversed
        // and the click is what you wait behind.
        Alert alert = wait.until(ExpectedConditions.alertIsPresent());
        assertTrue(alert.getText().contains("nothing here to locate"));
        alert.accept();
        waitForText("dialog-result", "alert acknowledged");

        // Nothing to locate, quite literally: the message the test just read is
        // not rendered anywhere. Note that it IS in the page source, as the
        // argument to the alert call in an inline script -- so grepping the
        // source would have "found" it and proved nothing. Checking has to
        // happen after the answer, too, because reading the DOM is an ordinary
        // command and an open dialog refuses every one of those.
        String rendered = (String) ((JavascriptExecutor) driver)
                .executeScript("return document.body.innerText;");
        assertFalse(rendered.contains("nothing here to locate"));
    }

    @Test
    void acceptingAndDismissingAConfirmAreDifferentFeatures() {
        open(PAGE);

        click("fire-confirm");
        wait.until(ExpectedConditions.alertIsPresent()).accept();
        waitForText("dialog-result", "confirm accepted");

        click("fire-confirm");
        wait.until(ExpectedConditions.alertIsPresent()).dismiss();

        // Same click, same page, opposite outcome. Which one happens is decided
        // entirely by the test, which is the only way it should be decided.
        waitForText("dialog-result", "confirm dismissed");
        waitForText("dialog-count", "2");
    }

    @Test
    void lettingTheDriverAnswerForYouIsTheTrap() {
        open(PAGE);
        click("fire-confirm");

        // The approach that looks right: click, then carry on reading the page.
        // Selenium's default unhandledPromptBehavior is "dismiss and notify",
        // so the driver answers the dialog itself and only tells you afterwards.
        // The notification is an exception on some later command -- which the
        // careless catch, or which lands in a test that was not about dialogs.
        UnhandledAlertException notified =
                assertThrows(UnhandledAlertException.class, () -> text("dialog-result"));
        assertTrue(notified.getMessage().contains("Delete everything?"));

        // And the page has already acted on an answer no test chose. Nothing
        // here is broken; the suite has simply been making decisions silently.
        waitForText("dialog-result", "confirm dismissed");
    }

    @Test
    void aPromptCarriesAValueBack() {
        open(PAGE);
        click("fire-prompt");

        Alert alert = wait.until(ExpectedConditions.alertIsPresent());
        assertEquals("What should the label say?", alert.getText());

        // A gap worth naming: Playwright can read the prompt's default value,
        // and WebDriver's Alert exposes only the message. The pre-filled "default
        // text" is unreadable from here, so a test that needs to assert on it has
        // to assert on what the page does with it instead.
        alert.sendKeys("typed by the test");
        alert.accept();

        waitForText("dialog-result", "prompt returned: typed by the test");
    }

    @Test
    void aChainedPairHasToBeAnsweredTwice() {
        open(PAGE);
        click("fire-chain");

        // Answering once leaves the second dialog blocking the page, and every
        // later command in the test fails for a reason that names neither
        // dialog. WebDriver has no dialog type to switch on either, so the
        // message is the only thing distinguishing the pair.
        Alert first = wait.until(ExpectedConditions.alertIsPresent());
        assertTrue(first.getText().contains("First of two"));
        first.accept();

        Alert second = wait.until(ExpectedConditions.alertIsPresent());
        assertTrue(second.getText().contains("And now the second"));
        second.accept();

        waitForText("dialog-result", "chain accepted");
    }

    @Test
    void aDialogCanArriveWhenNothingAskedForOne() {
        open(PAGE);
        click("fire-delayed");

        // Two seconds after the click resolved, with the test long since moved
        // on. Any ordinary command issued in that window is ambushed: it is the
        // command that reports the dialog, so the failure lands on whatever the
        // test happened to be doing rather than on the click that armed it.
        Alert late = wait.until(ExpectedConditions.alertIsPresent());
        assertTrue(late.getText().contains("nothing was waiting for this"));
        late.accept();

        waitForText("dialog-result", "delayed alert acknowledged");
    }

    @Test
    void aBeforeUnloadHandlerIsRegistered() {
        open(PAGE);

        // Checked by dispatching the event rather than by navigating away, for
        // the same reason the Playwright suite does it: whether the browser
        // actually raises the prompt depends on interaction heuristics that vary
        // by engine and by driver configuration. The handler's presence is the
        // contract; the prompt is the browser's decision.
        Object guarded = ((JavascriptExecutor) driver).executeScript(
                "const event = new Event('beforeunload', { cancelable: true });"
                        + "window.dispatchEvent(event);"
                        + "return event.returnValue !== '' || event.defaultPrevented;");
        assertEquals(Boolean.TRUE, guarded);
    }
}
