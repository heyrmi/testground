package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.List;
import java.util.Map;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.StaleElementReferenceException;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.support.ui.Select;
import org.openqa.selenium.support.ui.WebDriverWait;

/** /classic/cart — one cart, two frontends, and the proof that it was never in either of them. */
class CartTest extends Playground {

    private static final String CLASSIC = "/classic/cart";
    private static final String MODERN = "/app/checkout";

    @Test
    void theSameCartIsVisibleInTheZoneThatRunsNoJavaScript() {
        open(MODERN);
        addInTheModernZone("TG-MON-01");
        awaitText("cart-count", "1");

        open(CLASSIC);

        // A page that runs no script at all, and the cart is still there. That
        // is the only cheap proof that the state lives on the server rather than
        // in the component that rendered it.
        awaitText("cart-count", "1");
        assertEquals("£219.00", text("total"));
        assertEquals(1, cartLines("TG-MON-01").size());
    }

    @Test
    void andAChangeMadeThereShowsUpHere() {
        open(CLASSIC);
        new Select(find("field-sku")).selectByValue("TG-BAG-01");
        click("add-submit");

        // Submitting this form is a post and a redirect, and the click can
        // return while the replacement document is still arriving -- so the
        // count read here may belong to the page on its way out.
        awaitText("cart-count", "1");

        // Setting up through the cheap interface and asserting through the real
        // one. A form post and a redirect are far less to go wrong than driving
        // a React catalogue, and the state they leave behind is identical.
        open(MODERN);
        awaitText("cart-count", "1");
        assertEquals(1, cartLines("TG-BAG-01").size());
    }

    @Test
    void anOrderPlacedInOneZoneIsListedInTheOther() {
        open(MODERN);
        addInTheModernZone("TG-PAD-01");
        awaitText("cart-count", "1");
        click("go-to-payment");
        click("place-order");
        waitForPresent("order-number");

        // getDomProperty rather than getText: the order number is the one string
        // in this flow that exists nowhere else, and rendered text is whatever
        // CSS made of it rather than what the server minted.
        String number = find("order-number").getDomProperty("textContent").trim();
        assertTrue(number.matches("TG-\\d+"), "the order number was " + number);

        open(CLASSIC);
        assertEquals(
                1,
                driver.findElements(By.cssSelector("[data-testid='order-row'][data-order='" + number + "']")).size(),
                "the order placed in the modern zone was not listed here");

        // And the order emptied the cart in both zones at once, because there
        // was only ever one cart to empty.
        waitForPresent("cart-empty");
    }

    @Test
    void theScreenAndTheServerCanDisagreeAboutTheSameCart() {
        open(MODERN);
        awaitText("cart-count", "0");

        // A change made through the other zone, without touching this page.
        assertEquals(200L, addWithoutTheBrowser("TG-CAB-01").get("status"));

        // The trap the challenge is built around, and the screen is not lying:
        // it is showing the answer it fetched when it mounted, and nothing has
        // told it to ask again. A test that sets up through one interface and
        // asserts through a page it never reloaded is asserting on a cache.
        assertEquals("0", text("cart-count"));
        assertEquals(0, cartLines("TG-CAB-01").size());

        // Which is why the assertion has to be made against a page that has
        // actually asked the server since the change.
        driver.navigate().refresh();
        awaitText("cart-count", "1");
        assertEquals(1, cartLines("TG-CAB-01").size());
    }

    @Test
    void theZoneWithNoScriptRefusesWhatTheModernOneWouldNotOffer() {
        open(MODERN);

        // The modern zone refuses an out-of-stock product by never enabling the
        // button, which is a refusal a test can only observe as an absence.
        assertFalse(addButton("TG-HUB-01").isEnabled());

        // The zone with no script cannot disable anything it does not re-render,
        // so it refuses on the way in and says why.
        open(CLASSIC);
        new Select(find("field-sku")).selectByValue("TG-HUB-01");
        click("add-submit");

        awaitText("cart-message", "out of stock");
        waitForPresent("cart-empty");

        // The refusal is a 409 as well as a sentence, and the page will not tell
        // you that -- it rendered perfectly either way.
        assertEquals(409L, post(CLASSIC, "sku=TG-HUB-01").get("status"));
    }

    /**
     * Waits for a test id's text, tolerating the element being replaced while
     * the wait is running.
     *
     * <p>The base class's waitForText locates and reads in two steps and does
     * not ignore a stale reference, so a document swapped between them ends the
     * wait with an exception instead of another attempt. This challenge is where
     * that bites: the cart count exists on the page being left and on the page
     * arriving, so an ordinary wait can catch the outgoing one -- and the same
     * is true of a React re-render, which detaches the node it is replacing.
     * Ignoring staleness makes the retry mean what it looks like it means.
     */
    private void awaitText(String id, String expected) {
        new WebDriverWait(driver, TIMEOUT)
                .ignoring(StaleElementReferenceException.class)
                .until(d -> {
                    List<WebElement> found = d.findElements(testId(id));
                    return !found.isEmpty() && found.get(0).getText().trim().equals(expected);
                });
    }

    /** Adds a product through the React catalogue, which is the only place a product row exists. */
    private void addInTheModernZone(String sku) {
        addButton(sku).click();
    }

    /**
     * The add button belonging to one product row.
     *
     * <p>Narrowed by data-sku, as the challenge's manifest entry says to: the
     * test id names the kind of thing, and the data attribute picks which one.
     * Nothing here depends on the order the catalogue happens to come back in.
     */
    private WebElement addButton(String sku) {
        WebElement product = wait.until(driver -> {
            List<WebElement> rows =
                    driver.findElements(By.cssSelector("[data-testid='product'][data-sku='" + sku + "']"));
            return rows.isEmpty() ? null : rows.get(0);
        });
        return product.findElement(testId("add-to-cart"));
    }

    /** The cart lines for one product, in whichever zone is on screen. */
    private List<WebElement> cartLines(String sku) {
        return driver.findElements(By.cssSelector("[data-testid='cart-line'][data-sku='" + sku + "']"));
    }

    /**
     * Adds a product by posting the classic zone's form directly, leaving the
     * page on screen untouched.
     *
     * <p>Which is the point: it changes the server without giving the open page
     * any reason to notice, so the disagreement between the two is observable.
     */
    private Map<String, Object> addWithoutTheBrowser(String sku) {
        return post(CLASSIC, "sku=" + sku);
    }

    /**
     * Posts a form from the page and reports the status.
     *
     * <p>WebDriver cannot issue a request of its own and has no response object
     * to read a status from, so this runs in the page instead. Same-origin, so
     * it carries the pinned session cookie and lands in the same cart the
     * navigations see.
     */
    @SuppressWarnings("unchecked")
    private Map<String, Object> post(String path, String body) {
        driver.manage().timeouts().scriptTimeout(TIMEOUT);
        return (Map<String, Object>) ((JavascriptExecutor) driver).executeAsyncScript(
                "const [path, body, done] = arguments;"
                        + "fetch(path, {"
                        + "  method: 'POST',"
                        + "  headers: { 'content-type': 'application/x-www-form-urlencoded' },"
                        + "  body,"
                        + "})"
                        + "  .then(r => done({ status: r.status }))"
                        + "  .catch(e => done({ status: -1, error: String(e) }));",
                path, body);
    }
}
