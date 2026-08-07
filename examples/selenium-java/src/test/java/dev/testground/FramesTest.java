package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.Map;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.JavascriptExecutor;

/** /classic/frames — a locator searches one browsing context, and there are six of them here. */
class FramesTest extends Playground {

    private static final String PAGE = "/classic/frames";

    /** The session this class pins; the base class derives it from the class name. */
    private static final String SESSION = "se-FramesTest";

    @Test
    void aPageLevelSearchDoesNotLookInsideAFrame() {
        open(PAGE);

        // Every one of these elements exists and is on screen. None of them is
        // in this document, and a search that never leaves it finds nothing --
        // which reads exactly like an element that has not rendered yet, so the
        // instinct is to add a wait and watch it time out.
        assertEquals(0, count("embedded-target"));
        assertEquals(0, count("deepest-target"));
        assertEquals(0, count("cross-origin-target"));

        driver.switchTo().frame(find("frame-same-origin"));
        assertTrue(text("embedded-target").contains("Same-origin frame"));
    }

    @Test
    void theNestedChainIsDescendedOneFrameAtATime() {
        open(PAGE);

        // Playwright chains frameLocator calls, which reads as one expression.
        // WebDriver moves the whole session's context instead, so the same
        // descent is three statements and every locate afterwards is relative to
        // where it left off -- including find(), which is why nothing here needs
        // a frame-aware helper.
        driver.switchTo().frame(find("frame-nested"));
        driver.switchTo().frame(find("frame-level-2"));
        driver.switchTo().frame(find("frame-level-3"));

        assertTrue(text("deepest-target").contains("Three frames down"));
    }

    @Test
    void pageScriptCannotReadAcrossTheOriginBoundary() {
        open(PAGE);

        @SuppressWarnings("unchecked")
        Map<String, Object> reach = (Map<String, Object>) ((JavascriptExecutor) driver).executeScript(
                "const read = (frame) => {"
                        + "  try {"
                        + "    return frame.contentWindow.document.body !== null ? 'readable' : 'empty';"
                        + "  } catch (error) { return 'threw: ' + error.name; }"
                        + "};"
                        + "const same = document.querySelector('[data-testid=\"frame-same-origin\"]');"
                        + "const cross = document.querySelector('[data-testid=\"frame-cross-origin\"]');"
                        + "return { same: read(same), cross: read(cross), crossDocument: cross.contentDocument };");

        assertEquals("readable", reach.get("same"));
        assertTrue(
                String.valueOf(reach.get("cross")).startsWith("threw"),
                "the same-origin policy is not a wait problem, and no amount of retrying softens it");
        assertNull(reach.get("crossDocument"));
    }

    @Test
    void theDriverEntersTheCrossOriginFrameThatPageScriptCannot() {
        open(PAGE);

        // The whole lesson in one line. The refusal above is real, and it stops
        // anything running inside the page. WebDriver is not inside the page --
        // it is the other side of the browser -- so the boundary that beat
        // document.querySelector is not a boundary it has to cross.
        driver.switchTo().frame(find("frame-cross-origin"));

        assertTrue(text("cross-origin-target").contains("A different origin"));
    }

    @Test
    void bothOriginsResolveToTheSameSessionBecauseCookiesIgnoreThePort() {
        open(PAGE);

        // Not text(), which would answer SE-FRAMESTEST. getText reports what is
        // rendered and this label is uppercased by the stylesheet, so the
        // transform lands in the assertion. Playwright's text assertions read
        // textContent and never see it, which is why the same check is spelled
        // differently in the two suites -- and why a suite ported between them
        // fails on pages nobody changed.
        assertEquals(SESSION, textContent("parent-session"));

        driver.switchTo().frame(find("frame-cross-origin"));

        // Two origins, no shared DOM, one session. Cookies are scoped to a host
        // and the port is not part of that scope, so the playground_session
        // cookie this suite pins is sent to both listeners.
        assertEquals(SESSION, textContent("cross-origin-session"));
    }

    @Test
    void aSrcdocFrameHasContentButNoUrl() {
        open(PAGE);

        // getDomAttribute rather than getAttribute on purpose: the older call
        // falls back to the DOM property, and iframe.src reads as an empty
        // string whether the attribute is absent or empty. Asking for the
        // attribute itself is the only way to say "there is no src here".
        assertNull(find("frame-srcdoc").getDomAttribute("src"));

        driver.switchTo().frame(find("frame-srcdoc"));
        assertTrue(text("srcdoc-target").contains("arrived as an attribute"));
    }

    @Test
    void theSecondOriginReportsTheSessionDirectly() {
        open(PAGE);

        // Taken from the frame rather than hard-coded, because the server builds
        // it from the host the browser actually used and a fixed string would
        // only work on one machine.
        String whoami = find("frame-cross-origin").getDomAttribute("src").replace("/frame", "/whoami");

        // Navigated rather than fetched. A fetch from the parent page would be a
        // cross-origin request and the second listener sends no CORS headers, so
        // the read the browser allows is the one that leaves the page entirely.
        driver.get(whoami);

        assertTrue(
                driver.getPageSource().contains("\"session\":\"" + SESSION + "\""),
                "the second origin resolved a different session: " + driver.getPageSource());
        assertTrue(driver.getPageSource().contains("\"origin\":\"cross\""));
    }

    @Test
    void leavingAFrameIsAsExplicitAsEnteringOne() {
        open(PAGE);
        driver.switchTo().frame(find("frame-nested"));
        driver.switchTo().frame(find("frame-level-2"));

        // The context is a property of the session, not of a locator, so it
        // survives every command until something moves it back. Forgetting that
        // is how a later step in a long test looks for a top-level element and
        // is told, correctly, that it does not exist.
        assertEquals(0, count("parent-session"));

        driver.switchTo().parentFrame();
        assertEquals(0, count("parent-session"));

        driver.switchTo().defaultContent();
        assertEquals(SESSION, textContent("parent-session"));
    }

    /**
     * The text a node actually contains, before the stylesheet has had its say.
     *
     * <p>{@code getText} is rendered text: it honours text-transform, so a label
     * CSS has uppercased comes back uppercased and an assertion on a case
     * sensitive value -- a session id, an order number, a SKU -- fails on a page
     * that is perfectly correct.
     */
    private String textContent(String id) {
        return find(id).getDomProperty("textContent").trim();
    }
}
