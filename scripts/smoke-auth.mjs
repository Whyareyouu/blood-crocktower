import assert from "node:assert/strict";
import { randomBytes } from "node:crypto";

// Создаёт два тестовых аккаунта. Выполнять на локальном/тестовом стеке.
const web = process.env.SMOKE_WEB_URL || "http://localhost:3000";
const origin = process.env.PUBLIC_ORIGIN || new URL(web).origin;
const login = `smoke_${Date.now()}`;
const password = randomBytes(24).toString("base64url");

function device() {
  const cookies = new Map();
  let csrf = "";
  async function request(path, method = "GET", body, headers = {}) {
    const response = await fetch(new URL(`/api/v1${path}`, web), {
      method, headers: { Origin: origin, Cookie: [...cookies].map(([k, v]) => `${k}=${v}`).join("; "),
        "Content-Type": "application/json", "X-CSRF-Token": csrf, ...headers },
      body: body === undefined ? undefined : JSON.stringify(body), signal: AbortSignal.timeout(10_000), redirect: "error",
    });
    for (const cookie of response.headers.getSetCookie()) {
      const [pair] = cookie.split(";");
      const index = pair.indexOf("=");
      if (/Max-Age=0|Max-Age=-1/i.test(cookie)) cookies.delete(pair.slice(0, index));
      else cookies.set(pair.slice(0, index), pair.slice(index + 1));
    }
    return response;
  }
  return {
    request,
    async signIn(register = false, name = login) {
      const bootstrap = await request("/auth/csrf");
      assert.equal(bootstrap.status, 200);
      csrf = (await bootstrap.json()).csrfToken;
      const response = await request(`/auth/${register ? "register" : "login"}`, "POST", { login: name, password });
      assert.equal(response.status, register ? 201 : 200);
      const cookie = response.headers.getSetCookie().find((value) => value.startsWith("blood_session="));
      assert.match(cookie || "", /HttpOnly/i);
      assert.match(cookie, /SameSite=Strict/i);
      if (origin.startsWith("https:")) assert.match(cookie, /Secure/i);
      const session = await response.json();
      csrf = session.csrfToken;
      assert.deepEqual(Object.keys(session).sort(), ["csrfToken", "user"]);
      return session;
    },
  };
}

const first = device(), second = device(), outsider = device();
assert.equal((await first.request("/auth/session")).status, 401);
const created = await first.signIn(true);
assert.equal((await first.request("/auth/session")).status, 200);
const restored = await second.signIn();
assert.deepEqual(restored.user, created.user);
await outsider.signIn(true, `${login}_other`);
for (const current of [first, second, outsider]) {
  const response = await current.request("/users/me/rooms");
  assert.equal(response.status, 200);
  assert.deepEqual(await response.json(), { rooms: [] });
}
assert.equal((await first.request("/auth/logout", "POST", undefined, { "X-CSRF-Token": restored.csrfToken })).status, 403);
assert.equal((await first.request("/auth/logout", "POST", undefined, { Origin: "https://evil.example" })).status, 403);
assert.equal((await first.request("/auth/logout", "POST")).status, 204);
assert.equal((await first.request("/auth/session")).status, 401);
assert.equal((await second.request("/auth/session")).status, 200);
await second.request("/auth/logout", "POST");
await outsider.request("/auth/logout", "POST");
console.log("Проверены через Next.js: регистрация, два устройства, восстановление сессии, свои комнаты, CSRF, Origin и выход.");
