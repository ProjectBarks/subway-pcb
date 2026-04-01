import { defineConfig } from "vitest/config";

export default defineConfig({
	test: {
		include: ["ui/**/*.test.ts"],
		reporters: [
			"default",
			[
				"junit",
				{ outputFile: "../.test-results/frontend.xml", suiteName: "frontend" },
			],
		],
	},
});
