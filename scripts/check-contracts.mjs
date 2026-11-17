import assert from "node:assert/strict";
import { readFileSync, writeFileSync, mkdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname } from "node:path";
import { spawnSync } from "node:child_process";
import { createRequire } from "node:module";
import Ajv2020 from "ajv/dist/2020.js";
import addFormats from "ajv-formats";

const root = new URL("../", import.meta.url);
const read = (path) => JSON.parse(readFileSync(new URL(path, root), "utf8"));
const schema = read("docs/contracts/protocol.schema.json");
const api = read("docs/openapi.yaml");
const definitions = schema.$defs;
const nameOf = (ref) => ref.split("/").at(-1);

// Это проекция только используемых DTO, не универсальный генератор JSON Schema.
// Проверку ограничений выполняет готовый Ajv; неизвестный тип останавливает генерацию.
function typeOf(node, language) {
  if (node.$ref) return nameOf(node.$ref);
  if (node.oneOf) return language === "ts" ? node.oneOf.map((r) => typeOf(r, language)).join(" | ") : "";
  if (node.const !== undefined && language === "ts") return JSON.stringify(node.const);
  if (node.enum && language === "ts") return node.enum.map((v) => JSON.stringify(v)).join(" | ");
  switch (node.type) {
    case "string": return "string";
    case "integer": return language === "ts" ? "number" : "int";
    case "boolean": return language === "ts" ? "boolean" : "bool";
    case "array": return language === "ts" ? `Array<${typeOf(node.items, language)}>` : `[]${typeOf(node.items, language)}`;
    default: throw new Error(`Неподдерживаемая форма DTO: ${JSON.stringify(node)}`);
  }
}

const header = "// Сгенерировано из docs/contracts/protocol.schema.json. Не редактировать вручную.\n";
let ts = header;
let go = header + "package contracts\n\n";
for (const [name, node] of Object.entries(definitions)) {
  if (node.type === "object") {
    const empty = Object.keys(node.properties).length === 0;
    ts += empty ? `export type ${name} = Record<string, never>;\n\n` : `export interface ${name} {\n`;
    go += `type ${name} struct {\n`;
    for (const [property, propSchema] of Object.entries(node.properties)) {
      const optional = !node.required.includes(property);
      ts += `  ${property}${optional ? "?" : ""}: ${typeOf(propSchema, "ts")};\n`;
      const goName = property.replace(/Id$/, "ID").replace(/^./, (s) => s.toUpperCase());
      const goType = `${optional ? "*" : ""}${typeOf(propSchema, "go")}`;
      go += `${goName} ${goType} \`json:"${property}${optional ? ",omitempty" : ""}"\`\n`;
    }
    if (!empty) ts += "}\n\n";
    go += "}\n\n";
    if (node.properties.type?.const) {
      go += `const ${name}Type = ${JSON.stringify(node.properties.type.const)}\n\n`;
    }
  } else if (node.oneOf) {
    ts += `export type ${name} = ${typeOf(node, "ts")};\n\n`;
    go += `// ${name} объединяет допустимые направления сообщения. Декодирование и проверка констант обязательны в будущем транспорте.\n`;
    go += `type ${name} interface { is${name}() }\n\n`;
    for (const variant of node.oneOf) go += `func (${nameOf(variant.$ref)}) is${name}() {}\n`;
    go += "\n";
  } else {
    ts += `export type ${name} = ${typeOf(node, "ts")};\n\n`;
    go += `type ${name} ${typeOf(node, "go")}\n\n`;
    if (node.enum) {
      go += "const (\n";
      for (const value of node.enum) {
        const suffix = value.split("_").map((s) => s[0].toUpperCase() + s.slice(1)).join("");
        go += `${name}${suffix} ${name} = ${JSON.stringify(value)}\n`;
      }
      go += ")\n\n";
    }
  }
}
ts = ts.trimEnd() + "\n";
const formatted = spawnSync("gofmt", [], { input: go, encoding: "utf8", windowsHide: true });
assert.equal(formatted.status, 0, formatted.stderr || "Не удалось запустить gofmt");
for (const [path, content] of [["apps/web/lib/contracts/generated.ts", ts], ["apps/api/internal/contracts/generated.go", formatted.stdout]]) {
  const location = new URL(path, root);
  if (process.argv.includes("--write")) {
    mkdirSync(dirname(fileURLToPath(location)), { recursive: true });
    writeFileSync(location, content);
  } else assert.equal(readFileSync(location, "utf8"), content, `${path}: выполните node scripts/check-contracts.mjs --write`);
}

const ajv = new Ajv2020({ strict: true, allErrors: true });
addFormats(ajv);
ajv.addSchema(schema);
for (const name of Object.keys(definitions)) {
  assert.ok(ajv.getSchema(`${schema.$id}#/$defs/${name}`), `Схема ${name} должна компилироваться`);
}
const walkRefs = (value) => {
  if (!value || typeof value !== "object") return;
  if (value.$ref) {
    assert.ok(value.$ref.startsWith("./contracts/protocol.schema.json#/$defs/"), `Неизвестная ссылка ${value.$ref}`);
    assert.ok(definitions[nameOf(value.$ref)], `Нет DTO ${value.$ref}`);
  }
  for (const child of Object.values(value)) walkRefs(child);
};
walkRefs(api);
// Официальная OAS metaschema проверяет структуру документа; DTO отдельно Ajv выше.
// URI форматы здесь аннотации; локальные ссылки разрешаются walkRefs.
const openapiValidator = new Ajv2020({ strict: false, allErrors: true, validateFormats: false });
const openapiMeta = read('docs/contracts/openapi-3.1.schema.json');
// Ajv поддерживает dynamicRef только в корне ресурса. В официальной OAS схеме
// #meta объявлен внутри $defs/schema. В этой проверке диалект не переопределяется:
// привязываем ровно эти четыре ссылки к их статической цели, сохраняя исходник.
const bindMeta = (node) => {
  if (!node || typeof node !== 'object') return;
  if (node.$dynamicRef === '#meta') {
    delete node.$dynamicRef;
    node.$ref = '#/$defs/schema';
  }
  for (const value of Object.values(node)) bindMeta(value);
};
bindMeta(openapiMeta);
const validateOpenAPI = openapiValidator.compile(openapiMeta);
assert.ok(validateOpenAPI(api), JSON.stringify(validateOpenAPI.errors?.slice(0, 5)));
const ids = new Set();
let implemented = 0;
for (const [path, item] of Object.entries(api.paths)) {
  for (const [method, operation] of Object.entries(item)) {
    if (!['get', 'post', 'put', 'delete', 'patch'].includes(method)) continue;
    assert.ok(!ids.has(operation.operationId), `Повтор operationId ${operation.operationId}`);
    ids.add(operation.operationId);
    assert.ok(['implemented', 'planned'].includes(operation['x-implementation-status']));
    if (operation['x-implementation-status'] === 'implemented') {
      implemented++;
      assert.ok(['/health/live', '/health/ready', '/api/v1/status'].includes(path));
    }
    assert.ok(Object.keys(operation.responses).length);
    if (['post', 'put', 'delete', 'patch'].includes(method)) {
      assert.ok(operation.parameters.some((p) => p.name === 'X-CSRF-Token' && p.required));
    }
  }
}
assert.equal(implemented, 3);

for (const fixture of read('docs/contracts/fixtures.json')) {
  const validate = ajv.getSchema(`${schema.$id}#/$defs/${fixture.schema}`);
  assert.ok(validate, fixture.schema);
  assert.equal(validate(fixture.value), fixture.valid, `${fixture.name}: ${JSON.stringify(validate.errors)}`);
}

// TypeScript остаётся compile-time контрактом: regex/длины/границы проверяются Ajv,
// варианты сообщений и отсутствие скрытых полей проверяются ещё и компилятором.
const fixtures = read('docs/contracts/fixtures.json');
const typeErrors = new Set([
  'Внешний номер события запрещён', 'Тайная роль отсутствует в публичном DTO',
  'Гримуар отсутствует в публичном DTO', 'Неизвестная команда',
  'Неподдерживаемая версия', 'requestId обязателен',
  'Серверный ответ не клиентская команда', 'Чужое тело команды запрещено',
  'Числовая ревизия запрещена',
  'Примитив вместо пустого payload',
]);
const tsPath = fileURLToPath(new URL('apps/web/lib/contracts/generated.ts', root)).replaceAll('\\', '/');
const fixturePath = fileURLToPath(new URL('.cache/contracts-fixtures.ts', root));
mkdirSync(dirname(fixturePath), { recursive: true });
writeFileSync(fixturePath, fixtures.filter((f) => f.valid || typeErrors.has(f.name)).map((f) =>
  `${f.valid ? '' : '// @ts-expect-error: намеренно недопустимый DTO\n'}(${JSON.stringify(f.value)} satisfies import(${JSON.stringify(tsPath)}).${f.schema});`
).join('\n') + '\n');
const requireWeb = createRequire(new URL('apps/web/package.json', root));
const typescript = requireWeb('typescript');
const program = typescript.createProgram([fixturePath], { strict: true, noEmit: true, skipLibCheck: true, target: typescript.ScriptTarget.ES2020, types: [] });
const diagnostics = typescript.getPreEmitDiagnostics(program);
assert.equal(diagnostics.length, 0, typescript.formatDiagnosticsWithColorAndContext(diagnostics, {
  getCurrentDirectory: () => fileURLToPath(root), getCanonicalFileName: (f) => f, getNewLine: () => '\n',
}));
console.log(`Контракты проверены: ${Object.keys(definitions).length} DTO, ${ids.size} HTTP-операций, Go/TypeScript синхронизированы.`);
