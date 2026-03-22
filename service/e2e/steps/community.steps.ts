import { When, Then } from "@cucumber/cucumber";
import { PlaywrightWorld } from "../support/world";

Then("at least one plugin card should be visible", async function (this: PlaywrightWorld) {
  await this.page.waitForSelector("[data-preview-card]", { timeout: 15000 });
  const cards = await this.page.locator("[data-preview-card]").count();
  if (cards === 0) {
    throw new Error("No plugin cards found on community page");
  }
});

When("I search for {string} in the community", async function (this: PlaywrightWorld, query: string) {
  const responsePromise = this.page.waitForResponse(
    (response) => response.url().includes("/community/search"),
    { timeout: 10000 }
  );
  await this.page.locator('input[name="q"]').click();
  await this.page.locator('input[name="q"]').pressSequentially(query, { delay: 50 });
  await responsePromise;
});

When("I change the sort to {string}", async function (this: PlaywrightWorld, option: string) {
  const responsePromise = this.page.waitForResponse(
    (response) => response.url().includes("/community/search"),
    { timeout: 10000 }
  );
  await this.page.locator('select[name="sort"]').selectOption({ label: option });
  await responsePromise;
});
