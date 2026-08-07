package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.io.IOException;
import java.io.UncheckedIOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.text.Normalizer;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.stream.Stream;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import org.openqa.selenium.By;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.chrome.ChromeDriver;
import org.openqa.selenium.support.ui.ExpectedConditions;

/** /classic/downloads — a download is not a navigation, and Selenium has no event to wait for. */
class DownloadsTest extends Playground {

    private static final String PAGE = "/classic/downloads";

    /** Where Chrome is told to put whatever it saves, so the run leaves nothing behind. */
    @TempDir
    Path saved;

    /**
     * Points Chrome's downloads at this test's own directory.
     *
     * <p>This is where the two suites part company hardest. Playwright has a
     * download event that hands over the suggested filename and the finished
     * bytes; WebDriver has no concept of a download at all. The file lands on
     * the filesystem of whichever machine ran the browser and the test watches
     * for it, which is why the destination has to be arranged first -- otherwise
     * these tests would scatter files through the developer's Downloads folder.
     *
     * <p>Setting it needs the DevTools protocol, because it is a browser setting
     * rather than a page one. A superclass @BeforeEach runs first, so the driver
     * already exists by the time this does.
     */
    @BeforeEach
    void routeDownloadsIntoTheTemporaryDirectory() {
        ((ChromeDriver) driver).executeCdpCommand("Browser.setDownloadBehavior", Map.of(
                "behavior", "allow",
                "downloadPath", saved.toAbsolutePath().toString(),
                "eventsEnabled", true));
    }

    @Test
    void aDownloadIsNotANavigation() {
        open(PAGE);

        click("download-csv");

        // Waiting for the page to change would wait forever: the response is
        // never rendered and the address bar never moves. In Playwright the
        // thing to wait on is the download event; here there is no event, so the
        // only observable finish line is the file arriving on disk.
        assertTrue(driver.getCurrentUrl().endsWith(PAGE));
        assertEquals("report.csv", waitForSavedFile().getFileName().toString());
    }

    @Test
    void generatedContentIsByteIdenticalForASeed() {
        open(PAGE);

        String first = (String) fetch("/classic/downloads/report.csv?rows=5").get("text");
        String second = (String) fetch("/classic/downloads/report.csv?rows=5").get("text");

        assertEquals(first, second, "the same seed produced two different files");
        assertEquals("index,name,status,amount", first.split("\n")[0]);
        assertEquals(6, first.trim().split("\n").length);
    }

    @Test
    void theArchiveReallyIsAnArchive() {
        open(PAGE);
        Map<String, Object> response = fetch("/classic/downloads/bundle.zip");

        assertEquals("application/zip", response.get("type"));

        // "PK\x03\x04", the local file header every zip starts with. Asserting
        // the content type alone would pass on a server that mislabelled an
        // error page.
        assertEquals(List.of(0x50L, 0x4bL, 0x03L, 0x04L), head(response, 4));
    }

    @Test
    void theImageReallyIsAPng() {
        open(PAGE);
        Map<String, Object> response = fetch("/classic/downloads/pixel.png");

        // Byte 0 is 0x89, then the three ASCII letters of the signature.
        assertEquals(List.of(0x89L, (long) 'P', (long) 'N', (long) 'G'), head(response, 4));
    }

    @Test
    void anInlineFileIsRenderedRatherThanDownloaded() {
        open(PAGE);
        assertTrue(((String) fetch("/classic/downloads/notes.txt").get("disposition")).contains("inline"));

        // A different code path with a different failure: this one really is a
        // navigation, so the test that waits for a saved file waits forever and
        // the test that waits for the page is the one that works. Everything
        // about the link looks identical from the DOM.
        WebElement link = find("download-inline");
        link.click();
        wait.until(ExpectedConditions.stalenessOf(link));

        assertTrue(driver.findElement(By.tagName("body")).getText().contains("Served inline"));
        assertTrue(firstSavedFile(true).isEmpty(), "the browser saved a file it was told to render");
    }

    @Test
    void aNonAsciiFilenameTravelsInFilenameStarNotFilename() {
        open(PAGE);
        String disposition = (String) fetch("/classic/downloads/unicode.txt").get("disposition");

        // The plain filename parameter cannot carry these bytes, so the real
        // name is in a second, percent-encoded copy. Reading the obvious one
        // gives mojibake.
        assertTrue(disposition.contains("filename*=utf-8''"), disposition);
        assertTrue(disposition.contains("r%C3%A9sum%C3%A9"), disposition);

        click("download-unicode");

        // Playwright decodes it into suggestedFilename(). Selenium's equivalent
        // is whatever the browser wrote to disk -- which is the decoded name,
        // normalised by the filesystem rather than by the test, hence the
        // explicit normalisation before comparing.
        String onDisk = Normalizer.normalize(
                waitForSavedFile().getFileName().toString(), Normalizer.Form.NFC);
        assertTrue(onDisk.contains("résumé"), onDisk);
    }

    @Test
    void theGeneratedFileTakesItsTimeAndTheClickDoesNot() {
        open(PAGE);

        long started = System.currentTimeMillis();
        click("download-slow");
        Path file = waitForSavedFile();
        long elapsed = System.currentTimeMillis() - started;

        // The click returns the moment the request is sent. Three seconds of
        // generation happen afterwards, so a test that treated the click as the
        // end of the story would assert against a file that does not exist yet.
        assertTrue(elapsed > 2500, "the wait was on the click, not on the transfer: " + elapsed + "ms");
        assertEquals("slow-report.csv", file.getFileName().toString());
    }

    /**
     * Waits for Chrome to finish writing exactly one file into the download
     * directory and returns it.
     *
     * <p>A partial download is named with a .crdownload suffix until the last
     * byte lands, which is the only progress signal on offer: asserting on the
     * first filename that appears would read a file that is still being written.
     */
    private Path waitForSavedFile() {
        long deadline = System.currentTimeMillis() + TIMEOUT.toMillis();
        while (System.currentTimeMillis() < deadline) {
            Optional<Path> complete = firstSavedFile(false);
            if (complete.isPresent()) {
                return complete.get();
            }
            pause();
        }
        throw new AssertionError("nothing finished downloading within " + TIMEOUT);
    }

    /** The first file in the download directory, optionally counting partial ones. */
    private Optional<Path> firstSavedFile(boolean includePartial) {
        try (Stream<Path> entries = Files.list(saved)) {
            return entries
                    .filter(p -> includePartial || !p.getFileName().toString().endsWith(".crdownload"))
                    .findFirst();
        } catch (IOException e) {
            throw new UncheckedIOException(e);
        }
    }

    private void pause() {
        try {
            Thread.sleep(100);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new AssertionError("interrupted while waiting for a download", e);
        }
    }

    /**
     * Fetches a playground URL from page script and reports what came back.
     *
     * <p>Playwright hands its specs an HTTP client that shares the browser's
     * session. WebDriver has none, and a client built in Java would not carry
     * the cookie that pins this test's session -- so the request is issued from
     * the page itself, where the cookie is already in force and every response
     * header is readable because the origin matches.
     */
    @SuppressWarnings("unchecked")
    private Map<String, Object> fetch(String path) {
        driver.manage().timeouts().scriptTimeout(TIMEOUT);
        return (Map<String, Object>) ((JavascriptExecutor) driver).executeAsyncScript(
                "const done = arguments[arguments.length - 1];"
                        + "fetch(arguments[0]).then(async r => {"
                        + "  const bytes = new Uint8Array(await r.clone().arrayBuffer());"
                        + "  done({"
                        + "    status: r.status,"
                        + "    type: r.headers.get('content-type'),"
                        + "    disposition: r.headers.get('content-disposition'),"
                        + "    text: await r.text(),"
                        + "    head: Array.from(bytes.slice(0, 8)),"
                        + "  });"
                        + "}).catch(e => done({ error: String(e) }));",
                path);
    }

    @SuppressWarnings("unchecked")
    private List<Long> head(Map<String, Object> response, int bytes) {
        return ((List<Long>) response.get("head")).subList(0, bytes);
    }
}
