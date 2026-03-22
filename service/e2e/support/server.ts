import { execSync, ChildProcess, spawn } from "child_process";
import { rmSync, mkdirSync } from "fs";
import * as http from "http";
import * as path from "path";

const ROOT = path.resolve(__dirname, "../../..");
const DATA_DIR = path.join(ROOT, "service/tmp/e2e-data");

let serverProcess: ChildProcess | null = null;

export function buildServer(): void {
  console.log("Building Go server...");
  execSync("make backend/build", { cwd: ROOT, stdio: "inherit" });
}

export function startServer(): void {
  // Clean up any leftover data from previous runs
  try {
    rmSync(DATA_DIR, { recursive: true, force: true });
  } catch {
    // ignore cleanup errors
  }
  mkdirSync(DATA_DIR, { recursive: true });

  console.log("Starting Go server...");
  serverProcess = spawn(
    "./service/subway-server",
    [
      "--port", "8080",
      "--boards-dir", "service/public/boards",
      "--data-dir", "service/tmp/e2e-data",
      "--static-dir", "service/static",
    ],
    {
      cwd: ROOT,
      stdio: ["ignore", "pipe", "pipe"],
      env: { ...process.env },
    }
  );

  serverProcess.stdout?.on("data", (data: Buffer) => {
    if (process.env.DEBUG) console.log(`[server] ${data.toString().trim()}`);
  });

  serverProcess.stderr?.on("data", (data: Buffer) => {
    if (process.env.DEBUG) console.error(`[server] ${data.toString().trim()}`);
  });

  serverProcess.on("error", (err) => {
    console.error("Server process error:", err);
  });
}

export async function waitForHealth(
  baseURL = "http://localhost:8080",
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

export function stopServer(): void {
  if (serverProcess) {
    console.log("Stopping Go server...");
    serverProcess.kill("SIGTERM");
    serverProcess = null;
  }
  try {
    rmSync(DATA_DIR, { recursive: true, force: true });
  } catch {
    // ignore cleanup errors
  }
}
