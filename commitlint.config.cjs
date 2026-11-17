const hasRussian = (text) => /[А-Яа-яЁё]/u.test(text);

function russianDescriptions(parsed) {
  if (!hasRussian(parsed.subject || "")) return [false, "описание заголовка должно быть на русском языке"];
  for (const text of [parsed.body, parsed.footer]) {
    let code = false;
    for (const raw of (text || "").split("\n")) {
      const line = raw.trim();
      if (line.startsWith("```") || line.startsWith("~~~")) { code = !code; continue; }
      if (code || !line || hasRussian(line)) continue;
      if (/^(?:Co-authored-by|Signed-off-by): .+ <[^>]+>$/iu.test(line)) continue;
      if (/^(?:Refs|Closes|Fixes|Resolves):?\s+(?:#\d+|[\w.-]+\/[\w.-]+#\d+|https?:\/\/\S+)(?:[\s,]+(?:#\d+|[\w.-]+\/[\w.-]+#\d+))*$/iu.test(line)) continue;
      if (/^(?:https?:\/\/\S+|`[^`]+`)$/u.test(line)) continue;
      return [false, "описательный текст тела и footer должен быть на русском языке; код и служебные ссылки допускаются"];
    }
  }
  return [true];
}

module.exports = {
  extends: ["@commitlint/config-conventional"],
  defaultIgnores: false,
  plugins: [{ rules: { "russian-descriptions": russianDescriptions } }],
  rules: {
    "header-max-length": [2, "always", 100],
    "subject-case": [0],
    "body-max-line-length": [0],
    "footer-max-line-length": [0],
    "russian-descriptions": [2, "always"],
  },
};
