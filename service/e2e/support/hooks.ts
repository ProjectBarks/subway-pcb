import { BeforeAll, AfterAll, Before, After, setWorldConstructor, setDefaultTimeout, Status } from "@cucumber/cucumber";
import { PlaywrightWorld } from "./world";
import { waitForHealth } from "./server";

setDefaultTimeout(30000);
setWorldConstructor(PlaywrightWorld);

BeforeAll(async function () {
  // Server is started by the Makefile before cucumber launches.
  // Just verify it's reachable (fast check, already healthy).
  await waitForHealth();
});

AfterAll(async function () {
  // Server is stopped by the Makefile after cucumber exits.
});

Before(async function (this: PlaywrightWorld) {
  // Browser is opened in the viewport step or before first navigation
});

After(async function (this: PlaywrightWorld, scenario) {
  if (scenario.result?.status === Status.FAILED) {
    try {
      const png = await this.takeScreenshot(scenario.pickle.name);
      this.attach(png, "image/png");
    } catch {
      // screenshot may fail if browser never opened
    }
  }
  await this.closeBrowser();
});
