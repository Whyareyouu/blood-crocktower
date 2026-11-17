import { spawn } from "node:child_process";
import { existsSync, mkdirSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("../", import.meta.url));
const envFile = path.join(root, ".env");
if (existsSync(envFile)) process.loadEnvFile(envFile);

const env = {
  ...process.env,
  GOCACHE: process.env.GOCACHE || path.join(root, ".cache", "go-build"),
  GOMODCACHE: process.env.GOMODCACHE || path.join(root, ".cache", "gomod"),
};
mkdirSync(env.GOCACHE, { recursive: true });
mkdirSync(env.GOMODCACHE, { recursive: true });

const child = spawn("go", process.argv.slice(2), {
  cwd: path.join(root, "apps", "api"), env, stdio: "inherit",
});
child.on("error", (error) => {
  console.error(`Не удалось запустить Go: ${error.message}`);
  process.exitCode = 1;
});
child.on("exit", (code, signal) => {
  process.exitCode = signal ? 1 : (code ?? 1);
});
