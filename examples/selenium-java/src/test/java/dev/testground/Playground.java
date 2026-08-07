package dev.testground;

import java.time.Duration;
import java.util.List;

import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.openqa.selenium.By;
import org.openqa.selenium.Cookie;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.WebDriver;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.chrome.ChromeDriver;
import org.openqa.selenium.chrome.ChromeOptions;
import org.openqa.selenium.support.ui.ExpectedCondition;
import org.openqa.selenium.support.ui.WebDriverWait;

/**
 * The base every challenge test extends: a browser, an isolated session, and
 * the handful of helpers that keep the tests about the challenge rather than
 * about Selenium.
 *
 * <p>Isolation works differently here than in the Playwright suite, and the
 * difference is worth knowing. That suite pins a session with the
 * {@code X-Playground-Session} header, which WebDriver cannot set on ordinary
 * navigation. The playground accepts a {@code playground_session} cookie for
 * exactly this reason, so this suite pins the same session through the cookie
 * instead and gets the same isolation. Two test classes running in parallel
 * never see each other's server state.
 */
abstract class Playground {

    /** Where the playground is listening. Surefire passes it; the default matches `playground serve`. */
    private static final String BASE_URL =
            System.getProperty("playground.baseUrl", "http://127.0.0.1:7373");

    private static final boolean HEADLESS =
            Boolean.parseBoolean(System.getProperty("playground.headless", "true"));

    /**
     * How long a wait may retry before failing. Generous enough to survive a
     * loaded machine and short enough that a genuine hang is reported rather
     * than waited on.
     */
    protected static final Duration TIMEOUT = Duration.ofSeconds(10);

    protected WebDriver driver;
    protected WebDriverWait wait;

    @BeforeEach
    void openBrowser() {
        ChromeOptions options = new ChromeOptions();
        if (HEADLESS) {
            options.addArguments("--headless=new");
        }
        // A fixed window keeps layout-dependent challenges -- anything about
        // scrolling, stickiness or what is on screen -- from depending on
        // whatever size the machine happened to give us.
        options.addArguments("--window-size=1280,900", "--no-sandbox", "--disable-dev-shm-usage");

        driver = new ChromeDriver(options);
        wait = new WebDriverWait(driver, TIMEOUT);
        pinSession();
    }

    @AfterEach
    void closeBrowser() {
        if (driver != null) {
            driver.quit();
        }
    }

    /**
     * Gives this test its own copy of the playground, in its starting state.
     *
     * <p>A cookie cannot be set for a domain the browser is not on, so this
     * loads a cheap page first, replaces the session the server handed out with
     * one named after the test class, and leaves the browser there. Every later
     * navigation carries it.
     *
     * <p>The name is derived from the class rather than randomised, which makes
     * a failing run reproducible and a server-side session inspectable while
     * you debug it. The cost is that the session outlives the run, so state
     * from the last one would still be there: a task this suite completed an
     * hour ago is still complete. Resetting is what makes the id safe to reuse.
     */
    private void pinSession() {
        driver.get(BASE_URL + "/api/health");
        driver.manage().deleteAllCookies();
        driver.manage().addCookie(new Cookie("playground_session", sessionId(), "/"));
        resetSession();
    }

    /**
     * Puts the session back to its seeded state and clears every control-plane
     * rule.
     *
     * <p>Driven from page script because WebDriver has no way to issue a POST
     * of its own, and because it is a fair demonstration of the control plane:
     * the cookie is already set, so the request lands in this test's session
     * and nobody else's.
     */
    protected void resetSession() {
        driver.manage().timeouts().scriptTimeout(TIMEOUT);
        Object status = ((JavascriptExecutor) driver).executeAsyncScript(
                "const done = arguments[arguments.length - 1];"
                        + "fetch('/api/control/reset', { method: 'POST' })"
                        + "  .then(r => done(r.status))"
                        + "  .catch(e => done(String(e)));");
        if (!Long.valueOf(200L).equals(status)) {
            throw new IllegalStateException("could not reset the session, the playground answered " + status);
        }
    }

    /** A session id unique to this test class, in the character set the server accepts. */
    private String sessionId() {
        return "se-" + getClass().getSimpleName().replaceAll("[^A-Za-z0-9_-]", "");
    }

    /** Navigates to a playground path, for example {@code "/app/delayed-element"}. */
    protected void open(String path) {
        driver.get(BASE_URL + path);
    }

    /** The locator for a declared test id. Every challenge publishes these in its manifest entry. */
    protected static By testId(String id) {
        return By.cssSelector("[data-testid='" + id + "']");
    }

    /** Waits for the element to be present and returns it. */
    protected WebElement find(String id) {
        return wait.until(driver -> {
            List<WebElement> found = driver.findElements(testId(id));
            return found.isEmpty() ? null : found.get(0);
        });
    }

    /** Every element carrying a test id, without waiting: a page may legitimately render none. */
    protected List<WebElement> findAll(String id) {
        return driver.findElements(testId(id));
    }

    /** How many elements carry the test id right now. */
    protected int count(String id) {
        return findAll(id).size();
    }

    /**
     * Waits for an element's trimmed text to equal what is expected.
     *
     * <p>This is the shape almost every assertion here takes, and the reason is
     * the point of the playground: waiting for a condition retries until the
     * page settles, whereas reading the text once asserts on whichever render
     * happened to be current.
     */
    protected void waitForText(String id, String expected) {
        wait.until((ExpectedCondition<Boolean>) d -> {
            List<WebElement> found = d.findElements(testId(id));
            return !found.isEmpty() && found.get(0).getText().trim().equals(expected);
        });
    }

    /** Waits until at least one element with the test id exists. */
    protected void waitForPresent(String id) {
        wait.until((ExpectedCondition<Boolean>) d -> !d.findElements(testId(id)).isEmpty());
    }

    /** Waits until no element with the test id exists. */
    protected void waitForAbsent(String id) {
        wait.until((ExpectedCondition<Boolean>) d -> d.findElements(testId(id)).isEmpty());
    }

    /** The trimmed text of an element, once it is there. */
    protected String text(String id) {
        return find(id).getText().trim();
    }

    /** Clicks an element once it is there. */
    protected void click(String id) {
        find(id).click();
    }
}
