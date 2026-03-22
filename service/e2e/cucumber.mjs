export default {
  paths: ["e2e/features/**/*.feature"],
  requireModule: ["tsx/cjs"],
  require: ["e2e/steps/**/*.ts", "e2e/support/**/*.ts"],
  format: [
    "summary",
    "html:e2e/reports/report.html",
    "json:e2e/reports/report.json",
  ],
};
