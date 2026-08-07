package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.time.Duration;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.PageLoadStrategy;
import org.openqa.selenium.TimeoutException;
import org.openqa.selenium.WebDriver;
import org.openqa.selenium.chrome.ChromeDriver;
import org.openqa.selenium.chrome.ChromeOptions;

/** /classic/slow-pages — a server that has not answered and a page that never finishes are not the same wait. */
class SlowPagesTest extends Playground {

    private static final String HANGING = "/classic/slow-pages/hanging";

    @Test
    void theSlowPageIsAServerThatHasNotAnsweredYet() {
        long started = System.currentTimeMillis();
        open("/classic/slow-pages/slow?ms=1500");
        long elapsed = System.currentTimeMillis() - started;

        // Nothing was in the DOM to wait for, because nothing had been sent. The
        // only correct response to this one is patience.
        assertTrue(elapsed >= 1400, "the page answered in " + elapsed + "ms, so the delay was not served");
        waitForPresent("slow-body");
        assertEquals("1500", text("slow-ms"));
    }

    @Test
    void theDelayIsUnderTheCallerControl() {
        open("/classic/slow-pages/slow?ms=0");

        assertEquals("0", text("slow-ms"));
    }

    @Test
    void waitingForTheWholeLoadOnTheHangingPageIsAGuaranteedTimeout() {
        // The approach that looks right and is not. A driver made the ordinary
        // way loads pages with the "normal" ready state, which means get() sits
        // on the load event -- and one stylesheet on this page will never answer,
        // so that event never fires. The timeout is cut short here only so the
        // test costs three seconds rather than the default three hundred.
        driver.manage().timeouts().pageLoadTimeout(Duration.ofSeconds(3));

        assertThrows(
                TimeoutException.class,
                () -> open(HANGING),
                "the load event fired on a page with an outstanding subresource");

        // And this is the sting: the document was complete the whole time it was
        // being waited for. The wait failed on a page that was never not usable.
        waitForPresent("hanging-body");
        assertTrue(text("hanging-body").contains("usable right now"));
    }

    @Test
    void theReadyStateIsChosenWhenTheBrowserIsMadeRatherThanWhenAPageIsAskedFor() {
        // Where the two suites genuinely diverge. Playwright picks the ready
        // state per navigation -- goto(url, { waitUntil: 'domcontentloaded' }) --
        // so the same browser can be strict about one page and relaxed about the
        // next. In WebDriver it is a capability, fixed when the session is
        // created, so relaxing it means a second browser rather than a second
        // argument. Worth knowing before writing a suite that assumes otherwise.
        ChromeOptions options = new ChromeOptions();
        if (Boolean.parseBoolean(System.getProperty("playground.headless", "true"))) {
            options.addArguments("--headless=new");
        }
        options.addArguments("--window-size=1280,900", "--no-sandbox", "--disable-dev-shm-usage");
        options.setPageLoadStrategy(PageLoadStrategy.EAGER);

        // No session is pinned on this one: slow-pages keeps no server state, so
        // there is nothing for a second browser to collide with.
        WebDriver eager = new ChromeDriver(options);
        try {
            eager.manage().timeouts().pageLoadTimeout(Duration.ofSeconds(20));

            long started = System.currentTimeMillis();
            eager.get(System.getProperty("playground.baseUrl", "http://127.0.0.1:7373") + HANGING);
            long elapsed = System.currentTimeMillis() - started;

            // The same page, the same content, one capability different: it
            // returns as soon as the document is parsed instead of never.
            assertTrue(elapsed < 10_000, "eager still waited " + elapsed + "ms, so it waited for the load event");
            assertEquals(1, eager.findElements(testId("hanging-body")).size());
        } finally {
            eager.quit();
        }
    }
}
