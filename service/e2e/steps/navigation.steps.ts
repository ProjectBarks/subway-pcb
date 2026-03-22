import { Then } from "@cucumber/cucumber";
import { PlaywrightWorld } from "../support/world";
import { expect } from "@playwright/test";

Then("the page should have nav links to all sections", async function (this: PlaywrightWorld) {
  if (this.viewport === "mobile") {
    await this.page.locator("#mobile-menu-toggle").click();
    await this.page.waitForTimeout(300);
    for (const [text, href] of [["My Boards", "/boards"], ["Community", "/community"], ["Editor", "/editor"]]) {
      await expect(this.page.locator(`#mobile-menu a[href="${href}"]`).filter({ hasText: text })).toBeVisible();
    }
    await this.page.locator("#mobile-menu-toggle").click();
  } else {
    for (const [text, href] of [["My Boards", "/boards"], ["Community", "/community"], ["Editor", "/editor"]]) {
      await expect(this.page.locator(`nav a[href="${href}"]`).filter({ hasText: text }).first()).toBeVisible();
    }
  }
});

Then("the active nav link should be {string}", async function (this: PlaywrightWorld, linkText: string) {
  if (this.viewport === "mobile") {
    // On mobile, open the menu first to check active state
    await this.page.locator("#mobile-menu-toggle").click();
    await this.page.waitForTimeout(300);
    const activeLink = this.page.locator(`#mobile-menu a:has-text("${linkText}")`);
    await expect(activeLink).toHaveClass(/text-text-primary/);
    // Close menu
    await this.page.locator("#mobile-menu-toggle").click();
  } else {
    const activeLink = this.page.locator(`nav a:has-text("${linkText}")`).first();
    await expect(activeLink).toHaveClass(/text-text-primary/);
  }
});

Then("the mobile menu should behave correctly", async function (this: PlaywrightWorld) {
  if (this.viewport === "mobile") {
    // Menu should be hidden initially
    const menu = this.page.locator("#mobile-menu");
    await expect(menu).toHaveCSS("opacity", "0");

    // Click hamburger to open
    await this.page.locator("#mobile-menu-toggle").click();
    await this.page.waitForTimeout(300);
    await expect(menu).toHaveCSS("opacity", "1");

    // Click hamburger again to close
    await this.page.locator("#mobile-menu-toggle").click();
    await this.page.waitForTimeout(300);
    await expect(menu).toHaveCSS("opacity", "0");
  } else {
    // Desktop: nav links should be visible inline (no mobile menu needed)
    const navLinks = this.page.locator("nav a");
    const count = await navLinks.count();
    expect(count).toBeGreaterThan(0);
  }
});
