package dev.testground;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.awt.image.BufferedImage;
import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.io.UncheckedIOException;
import java.util.List;

import javax.imageio.ImageIO;

import org.junit.jupiter.api.Test;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.OutputType;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.support.ui.ExpectedCondition;

/** /app/visual-regression — a capture that is stable by construction, and the one pixel that proves the comparison is awake. */
class VisualRegressionTest extends Playground {

    private static final String PAGE = "/app/visual-regression";

    /**
     * How many differing pixels count as "the same picture".
     *
     * <p>Zero would be the honest answer and Chrome will not give it: the
     * rounded corners of the block and the swatch are re-rasterised between
     * paints, so an unchanged page differs from itself by a dozen or two pixels
     * along two curves. The number is stated here rather than buried in a
     * percentage because a tolerance you can name is one you can defend -- and
     * the test below is what stops it drifting upwards until nothing can fail.
     * For scale: the noise is under twenty pixels, and the one-pixel regression
     * this page offers moves about two thousand three hundred.
     */
    private static final int TOLERANCE = 100;

    @Test
    void aCaptureIsTheSameOnEveryRun() {
        open(PAGE);
        waitForText("freeze-state", "frozen");

        byte[] first = capture();
        driver.navigate().refresh();
        waitForText("freeze-state", "frozen");
        byte[] second = capture();

        int moved = differingPixels(first, second);
        assertTrue(moved <= TOLERANCE, "the same page drawn twice moved " + moved + " pixels");
    }

    @Test
    void andItIsNotTheSameWhenOnePixelChanges() {
        open(PAGE);
        waitForText("diff-state", "off");
        byte[] baseline = capture();

        open(PAGE + "?diff=1");
        waitForText("diff-state", "on");
        byte[] changed = capture();

        // The check worth more than any number of green runs. A comparison that
        // passes both ways is not comparing anything, and nothing in a passing
        // run distinguishes the two. Requiring the difference to clear the
        // tolerance by a wide margin is what keeps the tolerance honest.
        int moved = differingPixels(baseline, changed);
        assertTrue(
                moved > TOLERANCE * 5,
                "one element is a pixel wider and the comparison only noticed " + moved + " pixels");
    }

    @Test
    void theVolatileRegionIsMarkedRatherThanHidden() {
        open(PAGE);
        WebElement volatileRegion = find("volatile");

        assertEquals("true", volatileRegion.getDomAttribute("data-vr-mask"));
        assertTrue(volatileRegion.isDisplayed(), "masking is a comparison's problem, not a reason to hide the element");

        // It really does change ten times a second. Proving that in the DOM
        // rather than in the pixels is deliberate: how much of the picture a
        // tick moves depends on which digits happened to roll over, so an
        // assertion on that number would be as unstable as the thing it is
        // describing.
        waitForTextToChange("volatile", volatileRegion.getText().trim());
    }

    @Test
    void theMaskedRegionIsInsideThePictureAndWouldOtherwiseBeCompared() {
        open(PAGE);
        waitForText("freeze-state", "frozen");

        byte[] unmasked = find("reference").getScreenshotAs(OutputType.BYTES);
        byte[] masked = capture();

        // Same instant, same block, and hundreds of pixels apart: the clock is
        // inside the captured region, so without the mask every tick of it is a
        // difference. That is exactly the noise that drives people to raise the
        // tolerance until a real regression passes too.
        int moved = differingPixels(unmasked, masked);
        assertTrue(moved > TOLERANCE, "the mask covered nothing, so it was not the mask keeping the capture stable");
    }

    @Test
    void theAnimationIsFrozenUnlessAskedOtherwise() {
        open(PAGE);
        waitForText("freeze-state", "frozen");
        assertEquals("none", animationNameOf("spinner"), "a spinner mid-frame makes a capture differ from itself");

        open(PAGE + "?freeze=0");
        waitForText("freeze-state", "running");

        // Freeze rather than tolerate: with the animation running there is no
        // tolerance that both ignores the spinner and still catches a pixel.
        assertEquals("vr-spin", animationNameOf("spinner"));
    }

    @Test
    void theFlagArmsTheDifferenceWithoutRewritingAnyUrl() {
        open(PAGE);

        // A suite that already navigates to a hundred URLs cannot easily add
        // ?diff=1 to all of them, and the check only means anything if it is
        // actually run. The flag arms the same one pixel for a whole session.
        //
        // The page draws from its own query string; the state endpoint is the
        // single place that resolves both routes into one answer, so that is
        // where the flag is observed.
        assertFalse(stateSaysDiff(), "nothing should misbehave until it is asked to");

        setFeature("visual-regression.diff", true);
        assertTrue(stateSaysDiff());
    }

    /**
     * An element capture with the volatile regions masked.
     *
     * <p>Playwright takes a list of locators to mask and paints over them for
     * you. WebDriver has no such option, which makes this precisely the kind of
     * comparison the challenge is talking about: one that has to be told which
     * regions to ignore. Neutralising them generically from the published
     * attribute is the same thing Playwright does behind its own API, and is not
     * the same as pretending the element is absent.
     */
    private byte[] capture() {
        ((JavascriptExecutor) driver).executeScript(
                "document.querySelectorAll('[data-vr-mask]')"
                        + "  .forEach(el => { el.style.visibility = 'hidden'; });");
        return find("reference").getScreenshotAs(OutputType.BYTES);
    }

    /** How many pixels two captures disagree on, or every pixel if they are not even the same size. */
    private static int differingPixels(byte[] a, byte[] b) {
        BufferedImage left = decode(a);
        BufferedImage right = decode(b);
        if (left.getWidth() != right.getWidth() || left.getHeight() != right.getHeight()) {
            return Integer.MAX_VALUE;
        }

        int moved = 0;
        for (int y = 0; y < left.getHeight(); y++) {
            for (int x = 0; x < left.getWidth(); x++) {
                if (left.getRGB(x, y) != right.getRGB(x, y)) {
                    moved++;
                }
            }
        }
        return moved;
    }

    private static BufferedImage decode(byte[] png) {
        try {
            return ImageIO.read(new ByteArrayInputStream(png));
        } catch (IOException e) {
            throw new UncheckedIOException("the driver returned something that is not a PNG", e);
        }
    }

    /** The computed animation name, which is how a frozen element is told from a running one. */
    private String animationNameOf(String id) {
        return (String) ((JavascriptExecutor) driver)
                .executeScript("return getComputedStyle(arguments[0]).animationName;", find(id));
    }

    private void waitForTextToChange(String id, String before) {
        wait.until((ExpectedCondition<Boolean>) d -> {
            List<WebElement> found = d.findElements(testId(id));
            return !found.isEmpty() && !found.get(0).getText().trim().equals(before);
        });
    }

    /** Turns a control-plane feature flag on for this session. */
    private void setFeature(String flag, boolean enabled) {
        driver.manage().timeouts().scriptTimeout(TIMEOUT);
        Object status = ((JavascriptExecutor) driver).executeAsyncScript(
                "const done = arguments[arguments.length - 1];"
                        + "fetch('/api/control/feature', {"
                        + "  method: 'POST',"
                        + "  headers: { 'content-type': 'application/json' },"
                        + "  body: JSON.stringify({ flag: arguments[0], enabled: arguments[1] })"
                        + "}).then(r => done(r.status)).catch(e => done(String(e)));",
                flag, enabled);
        assertEquals(200L, status, "the control plane refused to set the flag");
    }

    /** Which way the state endpoint resolves the difference, whichever route asked for it. */
    private boolean stateSaysDiff() {
        driver.manage().timeouts().scriptTimeout(TIMEOUT);
        return Boolean.TRUE.equals(((JavascriptExecutor) driver).executeAsyncScript(
                "const done = arguments[arguments.length - 1];"
                        + "fetch('/api/app/visual-regression/state').then(r => r.json())"
                        + "  .then(b => done(b.diff)).catch(e => done(String(e)));"));
    }
}
