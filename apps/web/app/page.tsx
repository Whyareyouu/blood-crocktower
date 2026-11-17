export const dynamic = "force-dynamic";

async function isAPIReady(): Promise<boolean> {
  try {
    const base = process.env.API_INTERNAL_URL || "http://127.0.0.1:8080";
    const response = await fetch(`${base}/health/ready`, {
      cache: "no-store", signal: AbortSignal.timeout(2000),
    });
    return response.ok;
  } catch {
    return false;
  }
}

export default async function Home() {
  const ready = await isAPIReady();

  return (
    <main>
      <p className="eyebrow">Вранский Лес · Подготовка проекта</p>
      <h1>Кровь на<br />часовой башне</h1>
      <p className="intro">Каждая история начинается до первого удара часов.</p>
      <section aria-labelledby="status-title">
        <h2 id="status-title">Основа будущей игры</h2>
        <p>Каркас приложения подготовлен. Комнаты и игровые механики появятся на следующих этапах.</p>
        <p className="status"><span aria-hidden="true">{ready ? "●" : "○"}</span> {ready ? "API и база данных готовы" : "API или база данных пока недоступны"}</p>
        <form action="/" method="get">
          <button type="submit">Обновить состояние <span aria-hidden="true">↗</span></button>
        </form>
      </section>
      <footer>Next.js · Go · PostgreSQL</footer>
    </main>
  );
}
