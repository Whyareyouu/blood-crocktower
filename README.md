# Кровь на часовой башне

Веб-версия настольной игры. Сейчас подготовлен технический каркас:
Next.js App Router и TypeScript, Go API, PostgreSQL и миграции GORM.
Игровые механики, аккаунты, комнаты и WebSocket будут добавляться по
[дорожной карте](ROADMAP.md).

## Требования

- Node.js 24 и pnpm 10.12.4 (`corepack enable`, затем
  `corepack prepare pnpm@10.12.4 --activate`).
- Go 1.25 или новее для локальной разработки сервера.
- Docker Desktop с Linux containers и Compose v2 для БД или полного стека.

Все команды выполняются из корня репозитория. Зависимости JavaScript
зафиксированы в `pnpm-lock.yaml`, Go — в `apps/api/go.mod` и `go.sum`.

## Первый запуск

```powershell
Copy-Item .env.example .env
pnpm install --frozen-lockfile
pnpm run setup
```

В Linux/macOS вместо `Copy-Item` используется `cp .env.example .env`.
Пример содержит только локальный пароль разработки; реальные значения
хранятся в игнорируемом `.env`.

Полный стек:

```powershell
docker compose up --build
```

Открыть [приложение](http://localhost:3000) и
[готовность API](http://localhost:8080/health/ready).
Порядок запуска: PostgreSQL → одноразовая миграция → API → Next.js.
Данные PostgreSQL сохраняются в именованном томе. Остановка без удаления данных:

```powershell
docker compose down
```

## Локальная разработка

Запустить только PostgreSQL и применить миграцию:

```powershell
docker compose up -d db
pnpm db:migrate
```

В двух отдельных терминалах:

```powershell
pnpm dev:api
```

```powershell
pnpm dev:web
```

Корневые команды разработки читают `.env`. При изменении порта API обновите
также `API_INTERNAL_URL`. Кеши Go и pnpm находятся в игнорируемой `.cache/`.
Не запускайте локальные API/web одновременно с одноимёнными контейнерами
на тех же портах.

## Проверки

```powershell
pnpm check
pnpm build:web
pnpm build:api
```

Для интеграционного теста миграции и readiness нужен отдельный PostgreSQL
с настройками из `.env`. Тест создаёт только журнал миграций и не очищает БД:

```powershell
$env:TEST_DATABASE = '1'
pnpm test:api
Remove-Item Env:TEST_DATABASE
```

Без `TEST_DATABASE=1` интеграционная проверка явно пропускается.
CI отдельно проверяет клиент, Go с PostgreSQL и сборку всего Docker-стека.
Без настроенного remote CI ещё не запускается на сервере.

## Работа с Git

[Gitflow и Conventional Commits](docs/git-workflow.md). `main` содержит выпуски,
`develop` — интеграцию. Задачи выполняются в `feature/*`, выпуски — в `release/*`,
экстренные исправления — в `hotfix/*`. Слияния выполняются с `--no-ff`.

Сообщения коммитов, включая merge-коммиты, используют Conventional Commits
с русскими описаниями. `pnpm run setup` подключает локальный commit-msg hook;
команда не меняет глобальные настройки Git.

## Документация

- [Roadmap](ROADMAP.md).
- [Архитектура и проект WebSocket-контракта](docs/architecture.md).
- [Контракт реализованного HTTP API](docs/openapi.yaml).
- [Источники и покрытие правил](docs/rules/sources.md).
- [Проверка клиентских зависимостей](docs/dependencies.md).
- [Результаты проверки каркаса](docs/verification.md).

Compose предназначен для локального запуска: порты привязаны к `127.0.0.1`.
Для публичного размещения нужны собственные секреты, HTTPS/reverse proxy
и отдельная настройка доступности сервисов. Глобальный игровой администратор
и боевое размещение пока не реализованы.
