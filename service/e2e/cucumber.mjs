import os from "node:os";                                                                                      

const workers = Math.max(1, os.cpus().length - 1); 

export default {
  paths: ["e2e/features/**/*.feature"],
  requireModule: ["tsx/cjs"],
  require: ["e2e/steps/**/*.ts", "e2e/support/**/*.ts"],
  parallel: workers,
  format: [
    "summary",
    "html:e2e/reports/report.html",
    "json:e2e/reports/report.json",
  ],
};
