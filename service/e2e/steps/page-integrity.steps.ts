import { Then } from "@cucumber/cucumber";
import { PlaywrightWorld } from "../support/world";
import { expect } from "@playwright/test";

Then(
	"the page should have no horizontal overflow",
	async function (this: PlaywrightWorld) {
		const { scrollWidth, innerWidth } = await this.page.evaluate(() => ({
			scrollWidth: document.documentElement.scrollWidth,
			innerWidth: window.innerWidth,
		}));
		expect(
			scrollWidth,
			`Page overflows horizontally: scrollWidth=${scrollWidth} > innerWidth=${innerWidth}`,
		).toBeLessThanOrEqual(innerWidth);
	},
);

Then(
	"all local stylesheet and script sources should return 200",
	async function (this: PlaywrightWorld) {
		const urls: string[] = await this.page.evaluate(() => {
			const base = window.location.origin;
			const srcs: string[] = [];
			document
				.querySelectorAll('link[rel="stylesheet"][href]')
				.forEach((el) => {
					const href = (el as HTMLLinkElement).href;
					if (href.startsWith(base)) srcs.push(href);
				});
			document.querySelectorAll("script[src]").forEach((el) => {
				const src = (el as HTMLScriptElement).src;
				if (src.startsWith(base)) srcs.push(src);
			});
			return srcs;
		});

		expect(urls.length, "Expected at least one local asset").toBeGreaterThan(
			0,
		);

		const failures: string[] = [];
		for (const url of urls) {
			const resp = await this.page.request.get(url);
			if (resp.status() !== 200) {
				failures.push(`${url} returned ${resp.status()}`);
			}
		}
		expect(failures, `Broken assets:\n${failures.join("\n")}`).toEqual([]);
	},
);
