import { BeforeAll, AfterAll, Before, After, setWorldConstructor, setDefaultTimeout, Status } from "@cucumber/cucumber";
import { PlaywrightWorld } from "./world";
import { buildServer, startServer, waitForHealth, stopServer } from "./server";
import { mkdirSync } from "fs";

setDefaultTimeout(30000);
setWorldConstructor(PlaywrightWorld);

BeforeAll(async function () {
  mkdirSync("e2e/screenshots", { recursive: true });
  mkdirSync("e2e/reports", { recursive: true });
  buildServer();
  startServer();
  await waitForHealth();
});

AfterAll(async function () {
  stopServer();
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
