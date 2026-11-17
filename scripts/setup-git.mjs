import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("../", import.meta.url));
const settings = {
  "core.hooksPath": ".githooks",
  "commit.template": ".gitmessage",
  "gitflow.branch.master": "main",
  "gitflow.branch.develop": "develop",
  "gitflow.prefix.feature": "feature/",
  "gitflow.prefix.release": "release/",
  "gitflow.prefix.hotfix": "hotfix/",
  "gitflow.prefix.support": "support/",
  "gitflow.prefix.versiontag": "v",
};

try {
  execFileSync("git", ["rev-parse", "--git-dir"], { cwd: root, stdio: "pipe" });
  for (const [key, value] of Object.entries(settings)) {
    execFileSync("git", ["config", "--local", key, value], { cwd: root, stdio: "inherit" });
  }
  console.log("Локальные настройки Gitflow, шаблон коммита и commit-msg hook подключены.");
} catch {
  console.error("Не удалось настроить Git. Запустите pnpm run setup в доступном для записи клоне репозитория.");
  process.exitCode = 1;
}
