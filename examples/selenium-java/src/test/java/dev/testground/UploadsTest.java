package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Arrays;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import org.openqa.selenium.By;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.WebDriverException;
import org.openqa.selenium.WebElement;

/** /classic/uploads — a file input is set, never clicked, and every rule it advertises is advisory. */
class UploadsTest extends Playground {

    private static final String PAGE = "/classic/uploads";

    /**
     * Somewhere to put the files being uploaded.
     *
     * <p>Playwright can hand setInputFiles a buffer it invented, so its specs
     * never touch the disk. WebDriver has no such thing: the only value a file
     * input accepts is a path, so the file has to exist. JUnit's temporary
     * directory keeps that from leaking between runs.
     */
    @TempDir
    Path files;

    @Test
    void aFileInputIsSetThroughSendKeysAndNeverClicked() throws IOException {
        open(PAGE);

        // Clicking opens an operating-system picker nothing can drive. sendKeys
        // on a file input is not typing either -- the WebDriver specification
        // says the text is a path and the driver adds the file to the selection
        // -- which is why it works where clicking cannot.
        find("file-single").sendKeys(file("notes.txt", 64));
        clickAndWaitForReload("submit");

        waitForText("accepted-count", "1");
        assertEquals(1, count("upload-row"));
    }

    @Test
    void severalFilesGoInOneInputSeparatedByNewlines() throws IOException {
        open(PAGE);

        // Playwright takes an array. WebDriver takes one string, and the
        // separator is a newline -- which also means a path containing one
        // cannot be uploaded at all.
        find("file-multiple").sendKeys(String.join("\n",
                file("one.txt", 64),
                file("two.csv", 64),
                file("three.png", 64)));
        clickAndWaitForReload("submit");

        waitForText("accepted-count", "3");
        assertEquals(3, count("upload-row"));
    }

    @Test
    void sendKeysAppendsToTheSelectionRatherThanReplacingIt() throws IOException {
        open(PAGE);

        // The approach that looks right and is not. setInputFiles replaces the
        // selection, so a Playwright test can call it twice and get the second
        // file; sendKeys appends, so the same shape here uploads both. A helper
        // named "setFiles" wrapping sendKeys would quietly accumulate every file
        // a test ever touched, and the failure surfaces as a count nobody can
        // explain.
        find("file-multiple").sendKeys(file("first.txt", 64));
        find("file-multiple").sendKeys(file("second.txt", 64));
        clickAndWaitForReload("submit");

        waitForText("accepted-count", "2");

        // clear() is the reset the API does have: it empties the selection.
        open(PAGE);
        find("file-multiple").sendKeys(file("third.txt", 64));
        find("file-multiple").clear();
        find("file-multiple").sendKeys(file("fourth.txt", 64));
        clickAndWaitForReload("submit");

        waitForText("accepted-count", "1");
        assertEquals("fourth.txt", uploadRow("fourth.txt").getDomAttribute("data-name"));
    }

    @Test
    void acceptFiltersThePickerAndStopsNothing() throws IOException {
        open(PAGE);
        assertEquals(".png,.jpg,.jpeg", find("file-restricted").getDomAttribute("accept"));

        // The attribute says images only. The input takes a shell script without
        // a word of complaint, because accept narrows the picker's file list and
        // has no say over anything set any other way. Only the server's verdict
        // is real.
        find("file-restricted").sendKeys(file("script.sh", 64));
        clickAndWaitForReload("submit");

        waitForText("rejected-count", "1");
        assertTrue(uploadRow("script.sh").getText().contains("rejected"));
    }

    @Test
    void theSizeLimitFailsOnlyAfterTheWholeFileHasArrived() throws IOException {
        open(PAGE);

        find("file-single").sendKeys(file("big.txt", 100 * 1024));
        clickAndWaitForReload("submit");

        // Nothing rejects this early. The hundred kilobytes are transferred in
        // full, parsed, and only then refused -- so a test measuring "how long
        // does a rejected upload take" is measuring the upload, not the refusal.
        waitForText("rejected-count", "1");
        assertTrue(uploadRow("big.txt").getText().contains("larger than 65536 bytes"));
    }

    @Test
    void theServerReportsSizeAndTypeForWhatItReceived() throws IOException {
        open(PAGE);

        find("file-single").sendKeys(file("exact.csv", 1234));
        clickAndWaitForReload("submit");

        // The content type is the browser's guess from the extension, not
        // anything the test declared: Playwright's setInputFiles takes a
        // mimeType, sendKeys has nowhere to put one.
        waitForText("accepted-count", "1");
        String row = uploadRow("exact.csv").getText();
        assertTrue(row.contains("1234"), row);
        assertTrue(row.contains("text/csv"), row);
        assertTrue(row.contains("accepted"), row);
    }

    /** The result row the server rendered for one uploaded file. */
    private WebElement uploadRow(String name) {
        return find(By.cssSelector("[data-testid='upload-row'][data-name='" + name + "']"));
    }

    /** Writes a file of the given size and returns the absolute path a file input needs. */
    private String file(String name, int size) throws IOException {
        byte[] content = new byte[size];
        Arrays.fill(content, (byte) 'x');
        return Files.write(files.resolve(name), content).toAbsolutePath().toString();
    }

    /**
     * Clicks a control that replaces the document, and waits until it has.
     *
     * <p>Without this the next assertion races the reload. Playground#waitForText
     * locates an element and then reads it in a second round trip, so it can
     * find one in the outgoing document and read it after the incoming one has
     * arrived -- the stale-element failure this zone is about, landing in the
     * test instead of the product. Playwright's locators re-resolve and hide the
     * problem; in Selenium the synchronisation has to be written down.
     *
     * <p>ExpectedConditions.stalenessOf is the obvious way to write it and is
     * not quite enough: it treats only StaleElementReferenceException as the
     * signal, and a driver asked about an element exactly as the document is
     * being swapped answers with a raw protocol error instead. Both mean the
     * same thing -- the document that held this element has gone.
     */
    private void clickAndWaitForReload(String id) {
        clickAndWaitForReload(find(id));
    }

    private void clickAndWaitForReload(WebElement control) {
        WebElement outgoing = find("form");
        control.click();

        wait.until(d -> {
            try {
                outgoing.isDisplayed();
                return false;
            } catch (WebDriverException gone) {
                return true;
            }
        });
        // The old document has gone; this waits for the new one to finish
        // arriving, so the assertion after the call reads a settled page.
        wait.until(d -> "complete".equals(
                ((JavascriptExecutor) d).executeScript("return document.readyState")));
    }
}
