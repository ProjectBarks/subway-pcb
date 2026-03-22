import { When, Then } from "@cucumber/cucumber";
import { PlaywrightWorld } from "../support/world";
import * as assert from "assert";

interface HealthResponse {
  status: string;
  uptime: string;
  uptime_seconds: number;
  last_update: string;
  station_count: number;
}

let lastResponse: { status: number; body: HealthResponse };

When("I request the health endpoint", async function (this: PlaywrightWorld) {
  const res = await fetch(`${this.baseURL}/health`);
  const body = await res.json();
  lastResponse = { status: res.status, body };
});

Then("the response status should be {int}", async function (status: number) {
  assert.strictEqual(lastResponse.status, status);
});

Then("the response should contain {string}", async function (text: string) {
  assert.strictEqual(lastResponse.body.status, text);
});

Then("the response should have uptime_seconds greater than {int}", async function (min: number) {
  assert.ok(lastResponse.body.uptime_seconds > min, `uptime_seconds ${lastResponse.body.uptime_seconds} should be > ${min}`);
});

Then("the response should have station_count defined", async function () {
  assert.ok(lastResponse.body.station_count !== undefined, "station_count should be defined");
});
