package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.List;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.By;
import org.openqa.selenium.Cookie;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.WindowType;
import org.openqa.selenium.interactions.Actions;
import org.openqa.selenium.support.ui.ExpectedCondition;

/** /app/kanban — a pointer drag whose target is chosen before the board moves, a socket that changes the page with nothing to wait after, and an offline queue replayed against a board that did not wait for it. */
class KanbanTest extends Playground {

    private static final String PAGE = "/app/kanban";
    private static final String BOARD = "/api/app/kanban/board";
    private static final String MOVES = "/api/app/kanban/moves";

    @Test
    void aCardDraggedBetweenColumnsLandsWhereItWasAimed() {
        openBoard();
        assertEquals("todo", cardAttribute("card-1", "data-column"));

        dragOnto("card-1", "card-4");

        waitForCardAttribute("card-1", "data-column", "doing");
        waitForCardAttribute("card-1", "data-position", "0");
        waitForCardAttribute("card-4", "data-position", "1");
        waitForColumnCount("doing", "2");
        waitForContent("board-rev", "1");
        assertTrue(text("last-drop").contains("card-1 to doing 0"));
    }

    @Test
    void aMoveMadeThroughTheApiArrivesWithNothingToWaitAfter() {
        openBoard();
        waitForContent("watchers", "1");

        // Nothing on the page was touched. The only thing that will move this
        // card is a message the page did not ask for, so the wait has to be for
        // the board itself rather than for anything the test did.
        moveViaApi("card-2", "doing", 0);

        waitForCardAttribute("card-2", "data-column", "doing");
        waitForContent("last-event", "board");
        waitForContent("board-rev", "1");
    }

    @Test
    void presenceCountsTheTabsAndClosingOnePutsTheCountBack() {
        openBoard();
        waitForContent("watchers", "1");

        String first = driver.getWindowHandle();
        driver.switchTo().newWindow(WindowType.TAB);
        open(PAGE);
        String second = driver.getWindowHandle();

        driver.switchTo().window(first);
        waitForContent("watchers", "2");

        // The count is a fact the server publishes, so a departure is something
        // to wait for rather than something to sleep off.
        driver.switchTo().window(second);
        driver.close();
        driver.switchTo().window(first);
        waitForContent("watchers", "1");
    }

    /**
     * The composite's sharpest edge: the drop target is chosen on the last
     * pointer move and used on release, and the board is free to move in
     * between.
     */
    @Test
    void aCardArrivingMidDragSendsTheDropToAGapNobodyAimedAt() {
        openBoard();
        assertEquals("2", cardAttribute("card-3", "data-position"));

        // Aimed at the gap above card-2, which is where a release would land now.
        hold("card-3");
        travelTo(card("card-2"));
        waitForText("drop-target", "todo 1");

        // The other writer, driven from the test so the arrival is something
        // this test caused and can wait for. It runs from page script while the
        // button is still down, which does not disturb the drag: WebDriver's
        // input state lives on the remote end between perform() calls, so the
        // pointer is exactly where it was left.
        moveViaApi("card-4", "todo", 0);
        waitForPresent("board-changed");
        waitForCardAttribute("card-4", "data-column", "todo");

        release();

        // Index one now names the gap above card-1 rather than the gap above
        // card-2, because everything shifted down when card-4 arrived. The board
        // looks entirely plausible afterwards, which is why nothing reports it.
        waitForCardAttribute("card-4", "data-position", "0");
        waitForCardAttribute("card-3", "data-position", "1");
        waitForCardAttribute("card-1", "data-position", "2");
        waitForCardAttribute("card-2", "data-position", "3");
    }

    @Test
    void theBoardShownWhileOfflineIsALocalFictionAndTheFlushCorrectsIt() {
        openBoard();
        waitForContent("connection-state", "online");

        click("offline-toggle");
        waitForContent("connection-state", "offline");

        dragOnto("card-1", "card-4");
        dragOnto("card-2", "card-1");

        // The approach that looks like it works. Three cards really are in the
        // column, on screen, and this assertion passes -- it is just not an
        // assertion about the product.
        waitForColumnCount("doing", "3");
        waitForContent("queued-count", "2");
        assertEquals(2, count("queued-move"));

        assertEquals(
                "card-4",
                serverColumn("doing"),
                "the server never heard about either move");

        click("offline-toggle");
        waitForContent("connection-state", "online");

        // Replayed in order against the board as it is now. The first move fills
        // the column's last free place and the second has nowhere to go.
        assertTrue(text("flush-note").contains("1 of 2 queued moves applied"));
        assertEquals(1, count("refused-move"));
        assertTrue(narrow("refused-move", "card-2").getText().contains("at its limit"));

        waitForCardAttribute("card-1", "data-column", "doing");
        waitForCardAttribute("card-2", "data-column", "todo");
        waitForColumnCount("doing", "2");
    }

    @Test
    void theColumnLimitRefusesAThirdCardWhileOnlineAndSaysSo() {
        openBoard();

        moveViaApi("card-2", "doing", 0);
        waitForColumnCount("doing", "2");

        dragOnto("card-1", "card-2");

        waitForPresent("refusal");
        assertTrue(text("refusal").contains("at its limit"));
        assertEquals("todo", cardAttribute("card-1", "data-column"));
    }

    @Test
    void doneIsOneWayWhichIsADifferentRefusalFromAFullColumn() {
        openBoard();

        dragOnto("card-5", "card-1");

        waitForPresent("refusal");
        assertTrue(text("refusal").contains("one way"));
        assertEquals("done", cardAttribute("card-5", "data-column"));
        waitForContent("board-rev", "0");
    }

    @Test
    void twoWorkersHaveTheirOwnBoardAndTheirOwnPresenceCount() {
        openBoard();
        waitForContent("watchers", "1");

        neighbour("POST", MOVES, "{\"card\":\"card-1\",\"column\":\"doing\",\"index\":0}", "r.status");
        assertEquals(
                "0",
                neighbour("GET", BOARD, null, "b.board.watchers"),
                "a shared hub would make presence meaningless");

        // A shared board would have moved this card, and a shared hub would have
        // told this page about it.
        assertEquals("todo", cardAttribute("card-1", "data-column"));
        waitForContent("board-rev", "0");
        waitForContent("last-event", "presence");
    }

    // --- the drag ------------------------------------------------------------

    /**
     * Press on a card and keep holding it, which is where every drag here
     * starts.
     *
     * <p>Selenium has {@code Actions#dragAndDrop}, and it is the wrong tool for
     * this page twice over: it releases in the same sequence it pressed, so
     * there is no moment at which a test can let the board move underneath a
     * held card, and it aims at an element rather than at the gap between two.
     * Pressing, travelling and releasing as three separate calls is what makes
     * the mid-drag hazard reachable at all.
     */
    private void hold(String cardId) {
        new Actions(driver).moveToElement(card(cardId)).clickAndHold().perform();
    }

    /**
     * Travel to the target rather than jumping to it.
     *
     * <p>The page reads the drop position off the pointer's last move, so a
     * press and release with no move in between releases on the position it
     * started from and changes nothing at all.
     */
    private void travelTo(WebElement target) {
        new Actions(driver).moveToElement(target).perform();
    }

    private void release() {
        new Actions(driver).release().perform();
    }

    private void dragOnto(String cardId, String targetCardId) {
        hold(cardId);
        travelTo(card(targetCardId));
        release();
    }

    // --- helpers the base class does not carry -------------------------------

    /**
     * Loads the board and brings it on screen before anything is dragged.
     *
     * <p>The scroll is not incidental. The board sits below the description
     * panel, and a pointer action is aimed at a viewport coordinate: with the
     * cards off screen the move is dispatched to nothing, {@code
     * elementFromPoint} returns null, and the drag silently does nothing.
     * Scrolling once here rather than per drag also keeps the page still:
     * scrolling mid-drag would move the target out from under coordinates that
     * had already been measured.
     */
    private void openBoard() {
        open(PAGE);
        waitForPresent("card");
        ((JavascriptExecutor) driver).executeScript(
                "document.querySelector('[data-testid=\"column\"]').scrollIntoView({ block: 'center' });");
    }

    private static By cardBy(String id) {
        return By.cssSelector("[data-testid='card'][data-card-id='" + id + "']");
    }

    /** A card, re-found on every call: a card that changes column is a new node and a captured one would be stale. */
    private WebElement card(String id) {
        return wait.until(d -> {
            List<WebElement> found = d.findElements(cardBy(id));
            return found.isEmpty() ? null : found.get(0);
        });
    }

    /**
     * Where a card is, read from the card rather than from where it sits in a
     * list located earlier. {@code data-column} and {@code data-position} are
     * the published contract precisely because a board that moved underneath a
     * test makes any remembered ordering a lie.
     */
    private String cardAttribute(String cardId, String name) {
        return card(cardId).getDomAttribute(name);
    }

    private void waitForCardAttribute(String cardId, String name, String expected) {
        wait.until((ExpectedCondition<Boolean>) d -> expected.equals(cardAttribute(cardId, name)));
    }

    private void waitForColumnCount(String columnId, String expected) {
        By column = By.cssSelector("[data-testid='column'][data-column='" + columnId + "']");
        wait.until((ExpectedCondition<Boolean>) d -> {
            List<WebElement> found = d.findElements(column);
            if (found.isEmpty()) {
                return false;
            }
            List<WebElement> counts = found.get(0).findElements(testId("column-count"));
            return !counts.isEmpty() && counts.get(0).getDomProperty("textContent").trim().equals(expected);
        });
    }

    /** The one element carrying this test id and this {@code data-card}. */
    private WebElement narrow(String id, String card) {
        By by = By.cssSelector("[data-testid='" + id + "'][data-card='" + card + "']");
        return wait.until(d -> {
            List<WebElement> found = d.findElements(by);
            return found.isEmpty() ? null : found.get(0);
        });
    }

    /**
     * Waits for the text the DOM holds, which is not the text the page paints.
     *
     * <p>The connection state, the revision, the watcher count and the last
     * event all sit in the page's label style, which uppercases. Playwright's
     * {@code toHaveText} reads {@code textContent} and never sees that;
     * WebDriver's {@code getText} returns rendered text, so the base class's
     * {@code waitForText} would be comparing "ONLINE" with "online" and failing
     * on a stylesheet rather than on the board.
     */
    private void waitForContent(String id, String expected) {
        wait.until((ExpectedCondition<Boolean>) d -> {
            List<WebElement> found = d.findElements(testId(id));
            return !found.isEmpty() && expected.equals(found.get(0).getDomProperty("textContent").trim());
        });
    }

    /** Plays the other writer through the move endpoint, so the arrival is something this test caused rather than something it hopes has happened. */
    private void moveViaApi(String card, String column, int index) {
        assertEquals(
                "200",
                api("POST", MOVES,
                        "{\"card\":\"" + card + "\",\"column\":\"" + column + "\",\"index\":" + index + "}",
                        "r.status"),
                "the other writer's move was refused");
    }

    /** The cards in one column as the server holds them, which is not always the board on screen. */
    private String serverColumn(String id) {
        return api("GET", BOARD, null,
                "b.board.columns.find(c => c.id === '" + id + "').cards.map(c => c.id).join(',')");
    }

    /**
     * The same request from a session this test does not otherwise use.
     *
     * <p>Playwright makes a second request context for this. WebDriver has one
     * cookie jar, so the neighbour is addressed by the header the playground
     * accepts instead -- and then the cookie has to be put back, because the
     * server echoes a {@code Set-Cookie} for whichever session it just served
     * and would otherwise have moved this whole tab into the neighbour's board.
     * That repair is the price of having one browser rather than two clients,
     * and forgetting it is a failure that surfaces several assertions later.
     */
    private String neighbour(String method, String path, String body, String expression) {
        String answer = api(method, path, body, expression, "se-kanban-neighbour");
        driver.manage().addCookie(new Cookie("playground_session", "se-KanbanTest", "/"));
        return answer;
    }

    private String api(String method, String path, String body, String expression) {
        return api(method, path, body, expression, null);
    }

    /**
     * Sends a request the page would not send, and hands back one value from the
     * answer as a string.
     *
     * <p>The Playwright suite has an HTTP client beside the page for this.
     * WebDriver has none, so the fetch is driven from page script -- the same
     * request from a different place, and the only one available while a pointer
     * is held down: navigating to an endpoint would end the drag this test is
     * in the middle of.
     */
    private String api(String method, String path, String body, String expression, String session) {
        driver.manage().timeouts().scriptTimeout(TIMEOUT);
        return (String) ((JavascriptExecutor) driver).executeAsyncScript(
                "const [method, path, body, session] = arguments;"
                        + "const done = arguments[arguments.length - 1];"
                        + "const headers = {};"
                        + "if (body !== null) headers['Content-Type'] = 'application/json';"
                        + "if (session !== null) headers['X-Playground-Session'] = session;"
                        + "fetch(path, body === null ? { method, headers } : { method, headers, body })"
                        + "  .then(async r => { const b = await r.json(); done(String(" + expression + ")); })"
                        + "  .catch(e => done('the request failed: ' + e));",
                method, path, body, session);
    }
}
