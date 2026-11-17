# Кровь на часовой башне

Веб-версия настольной игры. Технический фундамент: Next.js App Router и
TypeScript, Go API, PostgreSQL и миграции GORM, каталог правил и сетевые
контракты. Игровые механики, аккаунты, комнаты и WebSocket-транспорт добавляются по
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
pnpm smoke
```

Для интеграционного теста миграции и readiness нужен отдельный PostgreSQL
с настройками из `.env`. Тесты создают изолированные временные таблицы
журнала миграций и удаляют только их через GORM:

```powershell
$env:TEST_DATABASE = '1'
pnpm test:api
Remove-Item Env:TEST_DATABASE
```

Без `TEST_DATABASE=1` интеграционная проверка явно пропускается.
CI отдельно проверяет клиент, Go с PostgreSQL и сборку всего Docker-стека.
Без настроенного remote CI ещё не запускается на сервере.

`pnpm smoke` проверяет уже запущенный стек: готовность API/БД, HTTP-прокси,
состояние на странице и статический ресурс. Адреса можно задать через
`SMOKE_API_URL` и `SMOKE_WEB_URL` (по умолчанию порты 8080 и 3000).
`pnpm check:contracts` проверяет OpenAPI, общую JSON Schema, примеры допустимых
и запрещённых сообщений и соответствие типов Go/TypeScript;
`pnpm generate:contracts` обновляет типы из схемы. Генерация требует `gofmt`.
`pnpm check:rules` проверяет полноту каталога коробки, источники, ночной
порядок и связь с таблицей требований; `pnpm test:rules` проверяет, что
валидатор обнаруживает повреждённые записи. Обе команды входят в `pnpm check`.

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
- [REST API: реализованные и запланированные операции](docs/openapi.yaml).
- [Контракты, версии и границы WebSocket](docs/contracts/README.md).
- [Источники и покрытие правил](docs/rules/sources.md).
- [Каталог 99 персонажей коробки](docs/rules/catalog.json) и [таблица соответствия](docs/rules/traceability.md).
- [Проверка клиентских зависимостей](docs/dependencies.md).
- [Результаты проверки первого этапа](docs/verification.md).

Compose предназначен для локального запуска: порты привязаны к `127.0.0.1`.
Для публичного размещения нужны собственные секреты, HTTPS/reverse proxy
и отдельная настройка доступности сервисов. Глобальный игровой администратор
и боевое размещение пока не реализованы.
