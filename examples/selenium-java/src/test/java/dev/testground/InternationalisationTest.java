package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.text.Normalizer;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.JavascriptExecutor;

/** /app/internationalisation — the same page in five scripts, and the assertions that do not survive it. */
class InternationalisationTest extends Playground {

    private static final String PAGE = "/app/internationalisation";

    @Test
    void switchingLocaleChangesTheDirectionWhichAnythingPositionalDependsOn() {
        open(PAGE);
        assertEquals("ltr", find("locale-panel").getAttribute("data-dir"));

        String englishGreeting = text("greeting");
        click("locale-ar-EG");

        // Assert on the identifiers the panel publishes, not on the words in
        // it: the greeting below changed and the feature is working perfectly,
        // which is exactly how a prose assertion fails.
        assertEquals("rtl", find("locale-panel").getAttribute("data-dir"));
        assertEquals("ar-EG", find("locale-panel").getAttribute("data-locale"));
        assertNotEquals(englishGreeting, text("greeting"));
    }

    @Test
    void aNumberAssertionWrittenForOneLocaleFailsInAnother() {
        open(PAGE);
        String english = text("format-number");

        click("locale-de-DE");
        String german = text("format-number");

        // The same amount with the separators swapped. Neither string is wrong,
        // and a test that hard-coded either one reports a bug in the other.
        assertTrue(english.contains(","), english);
        assertTrue(german.contains("."), german);
        assertNotEquals(english, german);
    }

    @Test
    void theSameInstantReadsAsTwoDifferentDays() {
        open(PAGE);
        click("locale-en-GB");
        String british = text("format-date");

        click("locale-ja-JP");
        String japanese = text("format-date");

        assertNotEquals(british, japanese);
        assertTrue(british.startsWith("04"), "day first here, and month first for half the world: " + british);
    }

    @Test
    void currencyFollowsTheLocaleNotTheAmount() {
        open(PAGE);
        click("locale-en-GB");
        waitForTextContaining("format-currency", "£");

        click("locale-ja-JP");
        waitForTextContaining("format-currency", "￥");
    }

    @Test
    void aTranslatedLabelIsLongerWhichIsAFindingRatherThanALocatorProblem() {
        open(PAGE);
        int english = Integer.parseInt(text("label-length"));

        click("locale-de-DE");
        int german = Integer.parseInt(text("label-length"));

        // German runs about a third longer. When the button stops fitting, the
        // layout is the thing that is wrong, not the locator that found it.
        assertTrue(german > english, "German was " + german + " characters and English " + english);
    }

    /**
     * The quiet one: two strings that render identically and compare unequal, so
     * the failure reads as the page being wrong rather than the comparison.
     */
    @Test
    void twoStringsRenderIdenticallyAndCompareAsDifferent() {
        open(PAGE);

        waitForText("naive-equal", "false");
        waitForText("normalised-equal", "true");

        // Read the source text rather than the rendered text: on screen these
        // are one word twice.
        String composed = find("nfc").getDomProperty("textContent");
        String decomposed = find("nfd").getDomProperty("textContent");

        assertNotEquals(composed, decomposed);
        // Java normalises to the same forms the page does, so the fix travels
        // across the wire intact.
        assertEquals(
                Normalizer.normalize(composed, Normalizer.Form.NFC),
                Normalizer.normalize(decomposed, Normalizer.Form.NFC));
    }

    @Test
    void oneEmojiIsNeitherOneCharacterNorOneCodePoint() {
        open(PAGE);

        int length = Integer.parseInt(text("family-length"));
        int codepoints = Integer.parseInt(text("family-codepoints"));

        assertTrue(length > codepoints, "surrogate pairs make the string longer than the code points it holds");
        assertTrue(codepoints > 1);

        // Java counts it exactly as the page does, because both measure UTF-16.
        // Neither number is what a person would call one thing.
        String family = find("family").getDomProperty("textContent");
        assertEquals(length, family.length());
        assertEquals(codepoints, family.codePointCount(0, family.length()));

        // What a person would call one thing. Java's own BreakIterator predates
        // these sequences, so the browser's segmenter is the honest source here.
        assertEquals(
                1L,
                ((JavascriptExecutor) driver).executeScript(
                        "return [...new Intl.Segmenter('en', { granularity: 'grapheme' })"
                                + "  .segment(arguments[0].textContent)].length;",
                        find("family")));
    }

    @Test
    void pluralCategoriesAreNotTwoEverywhere() {
        open(PAGE);
        click("locale-en-GB");
        waitForText("plural-one", "one");
        waitForText("plural-two", "other");

        // Arabic has a category for exactly two, so a message built from an
        // English singular and plural has nowhere to put this sentence.
        click("locale-ar-EG");
        waitForText("plural-two", "two");
    }

    @Test
    void theInputRoundTripsANonLatinScript() {
        open(PAGE);
        click("locale-hi-IN");

        String devanagari = "नमस्ते";
        find("script-input").sendKeys(devanagari);

        waitForText("typed-back", devanagari);
        assertEquals(String.valueOf(devanagari.length()), text("typed-length"));
    }

    private void waitForTextContaining(String id, String fragment) {
        wait.until(d -> {
            var found = d.findElements(testId(id));
            return !found.isEmpty() && found.get(0).getText().contains(fragment);
        });
    }
}
