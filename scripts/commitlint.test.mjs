import assert from "node:assert/strict";
import test from "node:test";
import load from "@commitlint/load";
import lint from "@commitlint/lint";

const config = await load();
const options = {
  parserOpts: config.parserPreset?.parserOpts,
  plugins: config.plugins,
  defaultIgnores: false,
};

for (const message of [
  "feat(lobby): добавить список комнат",
  "fix(api): исправить проверку имени карты",
  "chore: слить ветку feature/lobby в develop",
  "feat(api)!: изменить формат ответа\n\nBREAKING CHANGE: заменить поле roomId на id",
  "fix(api): исправить ответ\n\nСкрыть внутренние сведения.\n\nRefs: #12",
  "docs: описать запуск\n\nДобавить команды.\n\n```sh\npnpm install\n```",
  "chore: обновить зависимости\n\nCo-authored-by: Developer <dev@example.com>",
]) {
  test(`допускает ${message.split("\n")[0]}`, async () => {
    const result = await lint(message, config.rules, options);
    assert.equal(result.valid, true, JSON.stringify(result.errors));
  });
}

for (const message of [
  "add project scaffolding",
  "feat(web): add lobby",
  "feat(web): добавить экран\n\nAdd a lobby.",
  "feat(api)!: изменить ответ\n\nBREAKING CHANGE: remove old field",
  "chore: обновить настройки\n\nОписание на русском.\nEnglish explanation.",
  "unknown: добавить экран",
  "Merge branch 'feature/lobby' into develop",
]) {
  test(`отклоняет ${message.split("\n")[0]}`, async () => {
    const result = await lint(message, config.rules, options);
    assert.equal(result.valid, false);
  });
}
