import assert from "node:assert/strict";

// Проверяет уже запущенный стек; не меняет БД и не запускает сервисы.
const api = process.env.SMOKE_API_URL || "http://127.0.0.1:8080";
const web = process.env.SMOKE_WEB_URL || "http://127.0.0.1:3000";

async function get(base, path) {
  const response = await fetch(new URL(path, base), {
    signal: AbortSignal.timeout(10_000), redirect: "error",
  });
  assert.equal(response.status, 200, `${path}: ожидался HTTP 200`);
  return response;
}

for (const path of ["/health/live", "/health/ready"]) {
  const response = await get(api, path);
  assert.match(response.headers.get("content-type") || "", /application\/json/);
  assert.equal(response.headers.get("cache-control"), "no-store");
  assert.deepEqual(await response.json(), { status: "ok" });
}

const expectedStatus = { service: "blood-api", stage: "bootstrap" };
assert.deepEqual(await (await get(api, "/api/v1/status")).json(), expectedStatus);
assert.deepEqual(await (await get(web, "/api/v1/status")).json(), expectedStatus);

const page = await get(web, "/");
assert.match(page.headers.get("content-type") || "", /text\/html/);
const html = await page.text();
assert.match(html, /API и база данных готовы/);
const asset = html.match(/(?:src|href)="([^" ]*\/_next\/static\/[^" ]+\.(?:js|css))"/);
assert.ok(asset, "страница должна подключать статический ресурс Next.js");
const staticResponse = await get(web, asset[1].replaceAll("&amp;", "&"));
assert.match(staticResponse.headers.get("content-type") || "", /javascript|text\/css/);
assert.ok((await staticResponse.arrayBuffer()).byteLength > 0, "статический ресурс пуст");
console.log("Проверено: PostgreSQL через GORM readiness, API, прокси Next.js, страница готовности и статический ресурс.");
