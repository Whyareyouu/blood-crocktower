import assert from 'node:assert/strict';
import test from 'node:test';
import { readFile } from 'node:fs/promises';
import { loadRules, validateCatalog, validateTraceability } from './check-rules.mjs';

const { catalog, reference } = await loadRules();
const traceability = await readFile(new URL('../docs/rules/traceability.md', import.meta.url), 'utf8');

test('Таблица покрытия содержит каждую роль, её источник и честный статус', () => {
  validateTraceability(catalog, traceability);
});

test('Таблица покрытия не может потерять требование или источник', () => {
  assert.throws(() => validateTraceability(catalog, traceability.replace('`ROLE-imp`', '`ROLE-unknown`')));
  assert.throws(() => validateTraceability(catalog, traceability.replaceAll('https://wiki.bloodontheclocktower.com/Imp', 'нет-источника')));
});

test('Снимки официальных данных подтверждают полный состав и ночной порядок', () => {
  assert.equal(validateCatalog(catalog, reference).traveller, 15);
});

for (const [name, mutate] of [
  ['Пропущенный персонаж', (copy) => copy.roles.pop()],
  ['Дубликат вместо другой роли', (copy) => { copy.roles[0] = copy.roles[1]; }],
  ['Экспериментальная роль вместо базовой', (copy) => { copy.roles[0].id = 'alchemist'; }],
  ['Перестановка ночных действий', (copy) => copy.nightOrder.otherNight.reverse()],
  ['Неверная индивидуальная позиция', (copy) => { copy.roles.find((r) => r.id === 'imp').night.otherNight = 1; }],
  ['Утерянная повторная метка', (copy) => { copy.roles.find((r) => r.id === 'mathematician').reminders.pop(); }],
  ['Пропущенное исключение', (copy) => { copy.roles[0].exceptionsRu = []; }],
  ['Ложное утверждение о готовности механики', (copy) => { copy.roles[0].implementation.status = 'verified'; }],
  ['Пропущенное особое взаимодействие', (copy) => copy.baseJinxes.pop()],
  ['Потеря связи с источником', (copy) => { copy.roles[0].source.datasetCommit = 'main'; }],
]) {
  test(`Валидатор отвергает: ${name}`, () => {
    const copy = structuredClone(catalog);
    mutate(copy);
    assert.throws(() => validateCatalog(copy, reference));
  });
}
