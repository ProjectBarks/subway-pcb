import { Given, When, Then } from "@cucumber/cucumber";
import { PlaywrightWorld } from "../support/world";
import { expect } from "@playwright/test";
import { assertCanvasHasColors, assertWebGLCanvasRendered } from "../support/canvas-utils";

// ─── Viewport ──────────────────────────────────────────

Given("I am using a {word} viewport", async function (this: PlaywrightWorld, viewport: string) {
  await this.openBrowser(viewport);
});

// ─── Navigation ────────────────────────────────────────

When("I navigate to {string}", async function (this: PlaywrightWorld, path: string) {
  await this.page.goto(`${this.baseURL}${path}`, { waitUntil: "load" });
});

When("I click {string} in the navigation bar", async function (this: PlaywrightWorld, linkText: string) {
  if (this.viewport === "mobile") {
    await this.page.locator("#mobile-menu-toggle").click();
    await this.page.waitForTimeout(300);
    await this.page.locator(`#mobile-menu a:has-text("${linkText}")`).click();
  } else {
    await this.page.locator(`nav a:has-text("${linkText}")`).first().click();
  }
  await this.page.waitForLoadState("load");
});

When("I click the link {string}", async function (this: PlaywrightWorld, text: string) {
  await this.page.locator(`a:has-text("${text}")`).first().click();
  await this.page.waitForLoadState("load");
});

When("I click the button {string}", async function (this: PlaywrightWorld, text: string) {
  await this.page.locator(`button:has-text("${text}"), [role="button"]:has-text("${text}")`).first().click();
});

// ─── Visibility & Content ──────────────────────────────

Then("I should see the heading {string}", async function (this: PlaywrightWorld, text: string) {
  await expect(this.page.locator("h1")).toContainText(text);
});

Then("I should see {string}", async function (this: PlaywrightWorld, text: string) {
  await expect(this.page.getByText(text).first()).toBeVisible();
});

Then("the element {string} should be visible", async function (this: PlaywrightWorld, selector: string) {
  await expect(this.page.locator(selector).first()).toBeVisible();
});

Then("the element {string} should contain text {string}", async function (this: PlaywrightWorld, selector: string, text: string) {
  await expect(this.page.locator(selector).first()).toContainText(text);
});

Then("I should see a link {string} to {string}", async function (this: PlaywrightWorld, text: string, href: string) {
  await expect(this.page.locator(`a[href="${href}"]`).filter({ hasText: text }).first()).toBeVisible();
});

// ─── URL Assertions ────────────────────────────────────

const PAGE_URLS: Record<string, string> = {
  dashboard: "/boards",
  community: "/community",
  editor: "/editor",
};

Then("I should be on the {word} page", async function (this: PlaywrightWorld, pageName: string) {
  const expected = PAGE_URLS[pageName];
  if (!expected) throw new Error(`Unknown page: ${pageName}`);
  await expect(this.page).toHaveURL(new RegExp(expected));
});

Then("the URL should match {string}", async function (this: PlaywrightWorld, pattern: string) {
  await expect(this.page).toHaveURL(new RegExp(pattern));
});

// ─── Attribute Assertions ──────────────────────────────

Then("the element {string} should have attribute {string} with value {string}",
  async function (this: PlaywrightWorld, selector: string, attr: string, value: string) {
    await expect(this.page.locator(selector).first()).toHaveAttribute(attr, value);
  }
);

// ─── Form Interactions ─────────────────────────────────

When("I type {string} into the {string} input", async function (this: PlaywrightWorld, text: string, name: string) {
  await this.page.locator(`input[name="${name}"]`).fill(text);
});

When("I select {string} from the {string} dropdown", async function (this: PlaywrightWorld, option: string, name: string) {
  await this.page.locator(`select[name="${name}"]`).selectOption({ label: option });
});

// ─── Waiting ───────────────────────────────────────────

When("I wait for a response to {string}", async function (this: PlaywrightWorld, path: string) {
  await this.page.waitForResponse((response) => response.url().includes(path), { timeout: 10000 });
});

When("I wait for the element {string} to have class {string}",
  async function (this: PlaywrightWorld, selector: string, className: string) {
    await this.page.waitForFunction(
      ({ sel, cls }) => {
        const el = document.querySelector(sel);
        return el?.classList.contains(cls);
      },
      { sel: selector, cls: className },
      { timeout: 15000 }
    );
  }
);

// ─── Canvas Validation ─────────────────────────────────

Then("the canvas previews should render colors", async function (this: PlaywrightWorld) {
  await this.page.waitForFunction(
    () => document.querySelectorAll("[data-preview-card].preview-ready").length > 0,
    {},
    { timeout: 30000 }
  );

  const cards = await this.page.locator("[data-preview-card].preview-ready").all();
  let anyHasColors = false;

  for (const card of cards) {
    const canvas = card.locator("canvas").first();
    if (await canvas.count() === 0) continue;
    const result = await assertCanvasHasColors(this.page, `[data-preview-card].preview-ready canvas`);
    if (result.hasColors) {
      anyHasColors = true;
      break;
    }
  }

  if (!anyHasColors) {
    throw new Error("No canvas previews rendered colored pixels");
  }
});

Then("the WebGL viewer should render content", async function (this: PlaywrightWorld) {
  await this.page.waitForSelector("#board-viewer canvas", { timeout: 15000 });
  await this.page.waitForTimeout(1000);
  const result = await assertWebGLCanvasRendered(this.page, "#board-viewer");
  if (!result.hasContent) {
    throw new Error(`WebGL viewer has no content: ${JSON.stringify(result)}`);
  }
});
