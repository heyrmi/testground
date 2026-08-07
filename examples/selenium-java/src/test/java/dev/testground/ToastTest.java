package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.support.ui.ExpectedCondition;

/** /app/toast — a portalled, self-removing element, and the counters that outlive it. */
class ToastTest extends Playground {

    @Test
    void theToastRendersOutsideTheAppRoot() {
        open("/app/toast?dismissMs=20000");
        click("show-toast");

        waitForPresent("toast");

        // Scoping the search to the React root is the mistake the page is built
        // to punish: the toast is portalled onto body, so a locator rooted at
        // the application container reports nothing and the failure reads as a
        // toast that was never raised.
        assertEquals(
                0,
                driver.findElement(By.id("root")).findElements(testId("toast")).size(),
                "the portal should not be inside the app root");
        assertEquals(1, driver.findElements(By.cssSelector("body > [data-testid='toast-region']")).size());
    }

    @Test
    void theToastLeavesTheDomOnItsOwn() {
        open("/app/toast?dismissMs=1000");
        click("show-toast");

        waitForPresent("toast");
        waitForAbsent("toast");
        waitForText("toast-last", "1");
    }

    @Test
    void readWhatYouNeedOffTheToastBeforeAnythingSlowHappens() {
        open("/app/toast?dismissMs=3000");
        click("show-toast");

        // Taken while the toast is still there. Anything slow between the click
        // and the read -- a screenshot, a navigation, another wait -- and this
        // line throws NoSuchElementException, which reads as a toast that never
        // appeared rather than as an assertion that arrived late.
        String message = text("toast");

        waitForAbsent("toast");
        assertEquals("Saved change #1", message);
        waitForText("toast-last", "1");
    }

    @Test
    void countersOutliveTheToastSoAssertAgainstThose() {
        open("/app/toast?dismissMs=500");

        click("show-toast");
        click("show-toast");
        waitForText("toast-live", "0");

        // The toasts are long gone; the fact that they happened is not. These
        // are the durable targets, and they are what an assertion about the
        // feature should be aimed at.
        waitForText("toast-count", "2");
        waitForText("toast-last", "2");
    }

    @Test
    void twoToastsMakeOneTestIdMatchTwoNodesAndSeleniumWillNotComplain() {
        open("/app/toast?dismissMs=20000");
        click("show-toast");
        click("show-toast");

        wait.until((ExpectedCondition<Boolean>) d -> d.findElements(testId("toast")).size() == 2);

        // Here the two suites behave differently and the Selenium half is the
        // more dangerous one. Playwright's strict mode refuses an ambiguous
        // locator and fails loudly; findElement quietly hands back the first
        // match in document order, so a test written against "the toast" asserts
        // on whichever one happened to be drawn first and never says it chose.
        assertEquals("Saved change #1", text("toast"), "findElement silently picked the first of two");

        // The fix is the same in both: narrow by the attribute that tells them
        // apart rather than reaching for the first match.
        assertEquals(
                "Saved change #2",
                find(By.cssSelector("[data-testid='toast'][data-sequence='2']")).getText().trim());
        assertEquals(1, driver.findElements(By.cssSelector("[data-testid='toast'][data-sequence='1']")).size());
    }

    @Test
    void theDwellIsUnderTheCallerControl() {
        open("/app/toast?dismissMs=100");
        waitForText("dismiss-ms", "100");

        click("show-toast");

        // Short enough that the toast can be gone before the very next command
        // reaches the browser, which is why the assertion is on the counter
        // rather than on the toast.
        waitForText("toast-last", "1");
    }
}
