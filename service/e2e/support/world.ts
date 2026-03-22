import { World, IWorldOptions } from "@cucumber/cucumber";
import { Browser, BrowserContext, Page, chromium } from "@playwright/test";

const VIEWPORTS: Record<string, { width: number; height: number }> = {
  desktop: { width: 1280, height: 720 },
  mobile: { width: 375, height: 667 },
};

export class PlaywrightWorld extends World {
  browser!: Browser;
  context!: BrowserContext;
  page!: Page;
  viewport: string = "desktop";

  constructor(options: IWorldOptions) {
    super(options);
  }

  get baseURL(): string {
    return process.env.BASE_URL || "http://localhost:8080";
  }

  async openBrowser(viewport?: string): Promise<void> {
    if (viewport) this.viewport = viewport;
    const headless = process.env.HEADED !== "true";
    this.browser = await chromium.launch({ headless });
    const vp = VIEWPORTS[this.viewport] || VIEWPORTS.desktop;
    this.context = await this.browser.newContext({ viewport: vp });
    this.page = await this.context.newPage();
  }

  async closeBrowser(): Promise<void> {
    if (this.page) await this.page.close().catch(() => {});
    if (this.context) await this.context.close().catch(() => {});
    if (this.browser) await this.browser.close().catch(() => {});
  }

  async takeScreenshot(name: string): Promise<Buffer> {
    const sanitized = name.replace(/[^a-z0-9-]/gi, "-").toLowerCase();
    const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
    const filename = `FAIL-${sanitized}-${timestamp}.png`;
    const path = `e2e/screenshots/${filename}`;
    const buffer = await this.page.screenshot({ fullPage: true, path });
    return buffer;
  }
}
