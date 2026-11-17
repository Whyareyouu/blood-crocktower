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
