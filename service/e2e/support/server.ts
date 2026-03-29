import * as http from "http";

const BASE_URL = process.env.BASE_URL || "http://localhost:8080";

/**
 * Wait for the Go server to be healthy.
 * The server is started by the Makefile before cucumber launches.
 */
export async function waitForHealth(
  baseURL = BASE_URL,
  timeoutMs = 30000,
  intervalMs = 500
): Promise<void> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    try {
      const ok = await new Promise<boolean>((resolve) => {
        http
          .get(`${baseURL}/health`, (res) => {
            resolve(res.statusCode === 200);
            res.resume();
          })
          .on("error", () => resolve(false));
      });
      if (ok) {
        console.log("Server is healthy");
        return;
      }
    } catch {
      // retry
    }
    await new Promise((r) => setTimeout(r, intervalMs));
  }
  throw new Error(`Server did not become healthy within ${timeoutMs}ms`);
}
