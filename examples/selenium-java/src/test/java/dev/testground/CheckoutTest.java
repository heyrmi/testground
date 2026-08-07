package dev.testground;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.List;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.Cookie;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.Keys;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.support.ui.ExpectedCondition;
import org.openqa.selenium.support.ui.Select;

/** /app/checkout — a five-step flow where every total is revalidated and nothing may be carried forward. */
class CheckoutTest extends Playground {

    private static final String PAGE = "/app/checkout";

    @Test
    void theWholeFlowEndToEnd() {
        open(PAGE);

        addToCart("TG-PAD-01");
        waitForText("cart-count", "1");
        waitForText("subtotal", "£24.00");
        waitForText("shipping", "£4.99");

        fill("coupon-code", "SAVE10");
        click("apply-coupon");
        waitForText("discount", "£2.40");
        waitForText("total", "£26.59");

        click("go-to-payment");
        waitForSourceText("step", "pay");

        fill("checkout-card", "4242424242424242");
        click("place-order");

        waitForSourceText("step", "done");
        // The order number is the only lasting record the flow produced, so it
        // is read here rather than after navigating anywhere else.
        assertTrue(
                text("order-number").matches("TG-\\d+"),
                "the confirmation carried no order number in the published shape");
    }

    @Test
    void filtersCombineAndOutOfStockCannotBeAdded() {
        open(PAGE);

        new Select(find("category")).selectByValue("cables");
        waitForCount("product", 2);

        // The search narrows what the category already narrowed rather than
        // replacing it: asserting one product only holds if both applied.
        fill("search", "adapter");
        waitForCount("product", 1);

        fill("search", "");
        new Select(find("category")).selectByValue("peripherals");
        WebElement soldOut = waitForClickable(inProduct("TG-HUB-01", "add-to-cart"));
        assertFalse(soldOut.isEnabled(), "a product with no stock offered to be added anyway");
    }

    /**
     * The composite's sharpest edge: a total captured after applying a coupon
     * is a total nobody will be charged once the cart shrinks under it.
     */
    @Test
    void aCouponStopsApplyingWhenTheCartShrinksUnderIt() {
        open(PAGE);
        addToCart("TG-MON-01");

        fill("coupon-code", "BIGSPEND");
        click("apply-coupon");
        waitForText("discount", "£43.80");

        // Carrying that £43.80 forward is the mistake the challenge is built
        // around. The cart is now rebuilt out of a cheap line, and the discount
        // is re-read rather than remembered.
        waitForClickable(inCartLine("TG-MON-01", "remove-line")).click();
        addToCart("TG-CAB-01");

        waitForText("discount", "£0.00");
        assertTrue(
                text("coupon-note").contains("no longer applies"),
                "the coupon was dropped without saying so, which would hide the change from a reader");
    }

    @Test
    void aRefusedCouponSaysWhichRefusalItWas() {
        open(PAGE);
        addToCart("TG-PAD-01");

        // Three different reasons to say no. A test that only asserted "the
        // coupon was refused" would pass on all three and catch none of them.
        for (String[] attempt : List.of(
                new String[] {"NOPE", "no such coupon"},
                new String[] {"LASTYEAR", "expired"},
                new String[] {"BIGSPEND", "below this coupon"})) {
            fill("coupon-code", attempt[0]);
            click("apply-coupon");
            waitForTextContaining("coupon-note", attempt[1]);
        }
    }

    @Test
    void thePaymentOutcomeIsChosenNotDiscovered() {
        open(PAGE);
        addToCart("TG-PAD-01");
        click("go-to-payment");

        fill("checkout-card", "4000000000000002");
        click("place-order");
        waitForTextContaining("payment-error", "declined");

        fill("checkout-card", "4000000000009995");
        click("place-order");
        waitForTextContaining("payment-error", "insufficient funds");

        fill("checkout-card", "4242424242424242");
        click("place-order");
        waitForPresent("order-number");
    }

    @Test
    void placingAnOrderEmptiesTheCartSoAStalePageCannotChargeTwice() {
        open(PAGE);
        addToCart("TG-PAD-01");
        click("go-to-payment");
        click("place-order");
        waitForPresent("order-number");

        // Playwright has an APIRequestContext that shares the page's session;
        // WebDriver has no request client at all, so the second attempt is made
        // from page script. The session cookie is already on the browser, so it
        // lands in this test's cart -- the same session the order emptied.
        String answer = (String) ((JavascriptExecutor) driver).executeAsyncScript(
                "const done = arguments[arguments.length - 1];"
                        + "fetch('/api/app/shop/checkout', {"
                        + "  method: 'POST',"
                        + "  headers: { 'Content-Type': 'application/json' },"
                        + "  body: JSON.stringify({ email: 'buyer@example.test', card: '4242424242424242' })"
                        + "}).then(async r => done(r.status + '|' + (await r.json()).error))"
                        + " .catch(e => done('0|' + e));");

        // Correct behaviour that reads as a broken test until you know why.
        assertTrue(
                answer.startsWith("402|"),
                "a second checkout from a stale page was accepted, so an order can be placed twice: " + answer);
        assertTrue(answer.contains("nothing in the cart"), answer);
    }

    @Test
    void twoSessionsShopIndependently() {
        open(PAGE);
        addToCart("TG-KEY-01");
        waitForText("cart-count", "1");

        // A shared cart would make every parallel checkout test useless, so it
        // is proved rather than assumed: the same endpoint, read from a session
        // that has bought nothing.
        driver.manage().addCookie(new Cookie("playground_session", "se-checkout-neighbour", "/"));
        open("/api/app/shop/cart");
        assertTrue(
                driver.getPageSource().contains("\"count\":0"),
                "a neighbouring session saw this test's cart");
    }

    /** Clicks the add button belonging to one catalogue row. */
    private void addToCart(String sku) {
        waitForClickable(inProduct(sku, "add-to-cart")).click();
    }

    /**
     * The manifest publishes {@code data-sku} as the way to narrow a repeated
     * row, which is the only reason a compound selector appears here: Selenium
     * has no equivalent of Playwright's locator chaining, so the descendant is
     * expressed in one CSS query instead of two locates.
     */
    private static By inProduct(String sku, String childTestId) {
        return By.cssSelector("[data-testid='product'][data-sku='" + sku + "'] [data-testid='" + childTestId + "']");
    }

    private static By inCartLine(String sku, String childTestId) {
        return By.cssSelector("[data-testid='cart-line'][data-sku='" + sku + "'] [data-testid='" + childTestId + "']");
    }

    /**
     * Waits for the element to exist and be clickable. A disabled button never
     * becomes clickable, so callers that want to inspect a disabled control
     * take the element the wait returns rather than clicking it.
     */
    private WebElement waitForClickable(By by) {
        return wait.until(driver -> {
            List<WebElement> found = driver.findElements(by);
            return found.isEmpty() ? null : found.get(0);
        });
    }

    /**
     * Replaces a field's whole value the way a person would.
     *
     * <p>Playwright's {@code fill} is one call. The obvious Selenium spelling,
     * {@code clear()} then {@code sendKeys}, is not equivalent and quietly
     * breaks this page: {@code clear()} empties the input without the keystrokes
     * React listens for, so the box looks empty while the component still holds
     * the old text. Emptying the search box that way leaves the catalogue
     * filtered by a word that is no longer on screen. Backspacing over the
     * value sends real keys, which the page cannot miss.
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

    /** Waits until exactly this many elements carry the test id: filtering is a fetch, so it settles late. */
    private void waitForCount(String id, int expected) {
        wait.until((ExpectedCondition<Boolean>) d -> d.findElements(testId(id)).size() == expected);
    }

    /**
     * Waits for an element's source text, ignoring what CSS did to it.
     *
     * <p>The step label is styled uppercase, and {@code WebElement#getText}
     * reports what is on screen, so the base class's {@code waitForText} would
     * be comparing "PAY" against the "pay" the challenge documents. Playwright
     * never meets this because its text assertions read {@code textContent};
     * reading the DOM property is the same thing here.
     */
    private void waitForSourceText(String id, String expected) {
        wait.until((ExpectedCondition<Boolean>) d -> {
            List<WebElement> found = d.findElements(testId(id));
            return !found.isEmpty() && expected.equals(found.get(0).getDomProperty("textContent").trim());
        });
    }

    private void waitForTextContaining(String id, String fragment) {
        wait.until((ExpectedCondition<Boolean>) d -> {
            List<WebElement> found = d.findElements(testId(id));
            return !found.isEmpty() && found.get(0).getText().contains(fragment);
        });
    }
}
