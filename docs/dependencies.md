# Проверка клиентских зависимостей

Дата проверки: **17 ноября 2026 года**. Период статистики npm:
**9–15 ноября 2026 года** (`last-week`). Все прямые зависимости клиента
превышают установленный порог **100 000 скачиваний в неделю**.

| Пакет | Версия | Скачиваний за неделю | Назначение и источник |
| --- | --- | ---: | --- |
| `next` | 16.4.6 | 94 442 807 | App Router, [официальный проект](https://github.com/vercel/next.js), [npm API](https://api.npmjs.org/downloads/point/last-week/next) |
| `react` | 19.3.1 | 260 880 973 | Интерфейс, [официальный проект](https://github.com/facebook/react), [npm API](https://api.npmjs.org/downloads/point/last-week/react) |
| `react-dom` | 19.3.1 | 246 326 262 | Отрисовка React, [официальный проект](https://github.com/facebook/react), [npm API](https://api.npmjs.org/downloads/point/last-week/react-dom) |
| `@types/node` | 24.19.2 | 634 618 160 | Типы Node.js, [DefinitelyTyped](https://github.com/DefinitelyTyped/DefinitelyTyped), [npm API](https://api.npmjs.org/downloads/point/last-week/%40types%2Fnode) |
| `@types/react` | 19.3.0 | 235 009 646 | Типы React, [DefinitelyTyped](https://github.com/DefinitelyTyped/DefinitelyTyped), [npm API](https://api.npmjs.org/downloads/point/last-week/%40types%2Freact) |
| `@types/react-dom` | 19.3.0 | 199 488 063 | Типы React DOM, [DefinitelyTyped](https://github.com/DefinitelyTyped/DefinitelyTyped), [npm API](https://api.npmjs.org/downloads/point/last-week/%40types%2Freact-dom) |
| `eslint` | 9.39.5 | 207 051 018 | Статический анализ, [официальный проект](https://github.com/eslint/eslint), [npm API](https://api.npmjs.org/downloads/point/last-week/eslint) |
| `eslint-config-next` | 16.4.6 | 49 399 370 | Проверки Next.js и доступности, [официальный проект](https://github.com/vercel/next.js), [npm API](https://api.npmjs.org/downloads/point/last-week/eslint-config-next) |
| `typescript` | 5.9.3 | 427 220 745 | Проверка типов, [официальный проект](https://github.com/microsoft/TypeScript), [npm API](https://api.npmjs.org/downloads/point/last-week/typescript) |

ESLint 9 выбран из-за заявленной совместимости используемых плагинов Next.js:
у `eslint-plugin-import`, `eslint-plugin-jsx-a11y` и `eslint-plugin-react`
на момент проверки нет полной peer-совместимости с ESLint 10. Переход
на ESLint 10 выполняется вместе с совместимыми версиями плагинов.

Ссылки `last-week` возвращают подвижный период; при следующем добавлении
зависимости статистика проверяется заново. Числа выше — снимок указанного
периода, а не гарантия безопасности пакета. Условия выбора описаны
в `apps/web/AGENTS.md`.

## Инструменты проверки первого этапа

`ajv` **8.20.0** добавлен только в корневые `devDependencies`: он проверяет
JSON Schema 2020-12 и схемы OpenAPI, в клиентский bundle не входит.
Уже подключённые Next.js/TypeScript и GORM не заменяют валидатор схем;
собственная реализация JSON Schema не нужна.

Проверено 17 ноября 2026 года: **527 553 817** скачиваний за 9–15 ноября
по [npm Downloads API](https://api.npmjs.org/downloads/point/last-week/ajv).
Версия и [официальный репозиторий](https://github.com/ajv-validator/ajv)
подтверждены метаданными npm; возможности сверены с
[документацией Ajv](https://ajv.js.org/json-schema.html).
Точная версия закреплена в `package.json` и `pnpm-lock.yaml`.

`ajv-formats` **3.0.1** — корневое дополнение Ajv для проверки `date-time`
по RFC 3339. Оно заменяет недостаточную проверку даты регулярным выражением.
Официальные метаданные npm подтвердили версию и
[репозиторий](https://github.com/ajv-validator/ajv-formats).
На 17 ноября 2026 года [npm Downloads API](https://api.npmjs.org/downloads/point/last-week/ajv-formats)
показал **211 076 768** скачиваний за 10–16 ноября. Пакет не входит в bundle
клиента; точная версия сохранена в lockfile.

Каталог правил и smoke-проверка используют встроенные средства Node.js.
Миграции переиспользуют GORM и PostgreSQL driver; достаточность проверена по
[документации миграций](https://gorm.io/docs/migration.html) и
[транзакций](https://gorm.io/docs/transactions.html).

## Аккаунты и безопасность — этап 2

Проверено 17 ноября 2026 года: существующих аккаунтов, транспорта WebSocket
и политики доступа в проекте не было. Переиспользованы GORM, DTO,
генератор контрактов и React; новых прямых npm-зависимостей нет.
Для Go закреплены версии в `go.mod`/`go.sum`:

| Библиотека | Версия | Назначение и первичная документация |
| --- | --- | --- |
| `github.com/alexedwards/argon2id` | 1.0.0 | [Готовое создание и сравнение PHC-хешей](https://pkg.go.dev/github.com/alexedwards/argon2id); свой парсер хеша не нужен |
| `golang.org/x/crypto` | 0.53.0 | [Argon2id](https://pkg.go.dev/golang.org/x/crypto/argon2), транзитивная реализация; версия совместима с Go 1.25 |
| `github.com/coder/websocket` | 1.8.15 | [Upgrade, Origin, кадры, предел размера и закрытие](https://pkg.go.dev/github.com/coder/websocket@v1.8.15); свой WebSocket-протокол не пишется |
| `golang.org/x/time` | 0.14.0 | [Готовый token bucket](https://pkg.go.dev/golang.org/x/time/rate) для HTTP, аккаунтов и команд |
| `github.com/santhosh-tekuri/jsonschema/v6` | 6.0.2 | [Валидация JSON Schema 2020-12](https://pkg.go.dev/github.com/santhosh-tekuri/jsonschema/v6); переиспользована существующая схема |

Argon2id настроен по второму варианту [RFC 9106, раздел 4](https://www.rfc-editor.org/rfc/rfc9106.html#section-4):
64 МиБ, 3 прохода, 4 потока, соль 16 байт, ключ 32 байта. Одновременно
вычисляются максимум два хеша. Защита HTTP переиспользует
[`http.CrossOriginProtection`](https://pkg.go.dev/net/http#CrossOriginProtection)
из Go 1.25 и дополняется сессионным CSRF-токеном, точным `PUBLIC_ORIGIN`
и `SameSite=Strict`. Случайные токены создаются стандартным `crypto/rand`.
