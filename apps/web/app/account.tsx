"use client";

import { useEffect, useRef, useState } from "react";
import type { CSRFToken, RoomList, Session } from "../lib/contracts/generated";

class APIError extends Error {
  constructor(message: string, readonly status: number) { super(message); }
}

async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`/api/v1${path}`, {
    ...init, credentials: "same-origin", cache: "no-store",
  });
  if (!response.ok) {
    const body = await response.json().catch(() => null);
    throw new APIError(body?.message || "Сервис недоступен. Повторите попытку позже.", response.status);
  }
  return response.status === 204 ? undefined as T : response.json();
}

export default function Account() {
  const [session, setSession] = useState<Session | null>(null);
  const [rooms, setRooms] = useState<RoomList | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [register, setRegister] = useState(false);
  const [error, setError] = useState("");
  const [roomError, setRoomError] = useState("");
  const heading = useRef<HTMLHeadingElement>(null);

  useEffect(() => {
    const controller = new AbortController();
    async function restore() {
      try {
        const response = await fetch("/api/v1/auth/session", { cache: "no-store", signal: controller.signal });
        if (response.status === 401) return;
        if (!response.ok) throw new Error("Не удалось восстановить сессию. Обновите страницу.");
        setSession(await response.json());
      } catch (err) {
        if (!controller.signal.aborted) setError(err instanceof Error ? err.message : "Ошибка соединения.");
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    }
    void restore();
    return () => controller.abort();
  }, []);

  useEffect(() => {
    if (!session) return;
    const controller = new AbortController();
    api<RoomList>("/users/me/rooms", { signal: controller.signal }).then(setRooms).catch(() => {
      if (!controller.signal.aborted) setRoomError("Не удалось загрузить ваши комнаты. Обновите страницу.");
    });
    return () => controller.abort();
  }, [session]);

  async function authenticate(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    setBusy(true);
    setError("");
    try {
      const csrf = await api<CSRFToken>("/auth/csrf");
      const result = await api<Session>(register ? "/auth/register" : "/auth/login", {
        method: "POST", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf.csrfToken },
        body: JSON.stringify({ login: data.get("login"), password: data.get("password") }),
      });
      form.reset();
      setRooms(null);
      setRoomError("");
      setSession(result);
      requestAnimationFrame(() => heading.current?.focus());
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка соединения.");
    } finally { setBusy(false); }
  }

  async function logout() {
    if (!session) return;
    setBusy(true);
    setError("");
    try {
      // Актуальный токен учитывает вход/выход в соседней вкладке.
      const csrf = await api<CSRFToken>("/auth/csrf");
      await api<void>("/auth/logout", { method: "POST", headers: { "X-CSRF-Token": csrf.csrfToken } });
      setSession(null);
      setRooms(null);
      requestAnimationFrame(() => heading.current?.focus());
    } catch (err) {
      if (err instanceof APIError && err.status === 401) {
        setSession(null);
        setRooms(null);
      } else {
        setError(err instanceof Error ? err.message : "Ошибка соединения.");
      }
    } finally { setBusy(false); }
  }

  return (
    <section aria-labelledby="account-title" aria-busy={loading || busy}>
      <h2 id="account-title" ref={heading} tabIndex={-1}>{session ? `Вы вошли как ${session.user.login}` : "Ваш аккаунт"}</h2>
      {loading ? <p role="status">Восстанавливаем сессию…</p> : session ? <>
        <p>Аккаунт доступен на любом устройстве по вашему логину и паролю.</p>
        <button type="button" disabled={busy} onClick={logout}>{busy ? "Выходим…" : "Выйти из аккаунта"}</button>
        <h3>Мои комнаты</h3>
        {roomError ? <p role="alert">{roomError}</p> : rooms === null ? <p role="status">Загружаем комнаты…</p> : rooms.rooms.length === 0 ?
          <p>Вы пока не участвуете в комнатах. Создание комнат появится на следующем этапе.</p> :
          <ul>{rooms.rooms.map((room) => <li key={room.id}>{room.name} · {room.playerCount} из {room.capacity} игроков</li>)}</ul>}
      </> : <>
        <div className="account-modes" aria-label="Действие с аккаунтом">
          <button type="button" aria-pressed={!register} disabled={busy} onClick={() => { setRegister(false); setError(""); }}>Вход</button>
          <button type="button" aria-pressed={register} disabled={busy} onClick={() => { setRegister(true); setError(""); }}>Регистрация</button>
        </div>
        <form method="post" onSubmit={authenticate}>
          <label htmlFor="login">Логин</label>
          <input id="login" name="login" autoComplete="username" autoCapitalize="none" spellCheck={false} required
            minLength={3} maxLength={32} pattern="[A-Za-z0-9_\-]+" aria-describedby="login-help" disabled={busy} />
          <p id="login-help" className="field-help">3–32 латинских символа, цифры, _ или -. Регистр не учитывается.</p>
          <label htmlFor="password">Пароль</label>
          <input id="password" name="password" type="password" autoComplete={register ? "new-password" : "current-password"}
            required minLength={12} maxLength={128} aria-describedby="password-help" disabled={busy} />
          <p id="password-help" className="field-help">От 12 до 128 символов. Можно вставить пароль из менеджера паролей.</p>
          <button type="submit" disabled={busy}>{busy ? "Подождите…" : register ? "Создать аккаунт" : "Войти"}</button>
        </form>
      </>}
      {error && <p role="alert" className="error">{error}</p>}
    </section>
  );
}
