import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFile } from 'node:fs/promises';
import { pathToFileURL } from 'node:url';

const rulesRoot = new URL('../docs/rules/', import.meta.url);
const phases = ['firstNight', 'otherNight'];
const systemSteps = new Set(['dusk', 'minioninfo', 'demoninfo', 'dawn']);
const baseEditions = new Set(['tb', 'bmr', 'snv', 'fabled']);

export async function loadRules() {
  const catalog = JSON.parse(await readFile(new URL('catalog.json', rulesRoot), 'utf8'));
  const reference = {};
  for (const filename of ['roles.json', 'nightsheet.json', 'jinxes.json']) {
    const raw = await readFile(new URL(`upstream/${filename}`, rulesRoot));
    const source = catalog.upstream.files.find((file) => file.path === `resources/data/${filename}`);
    assert.ok(source, `Нет источника ${filename}`);
    assert.equal(createHash('sha256').update(raw).digest('hex'), source.sha256, `Изменён снимок ${filename}`);
    assert.equal(source.url, `https://raw.githubusercontent.com/ThePandemoniumInstitute/botc-release/${catalog.upstream.commit}/${source.path}`);
    reference[filename.replace('.json', '')] = JSON.parse(raw.toString('utf8').replace(/^\uFEFF/, ''));
  }
  return { catalog, reference };
}

function russianText(value, label) {
  assert.ok(typeof value === 'string' && /[А-Яа-яЁё]/u.test(value) && value.trim().length > 0, label);
}

export function validateCatalog(catalog, reference) {
  assert.equal(catalog.schemaVersion, 1);
  assert.match(catalog.upstream.commit, /^[a-f0-9]{40}$/u);
  const expected = reference.roles.filter((role) => baseEditions.has(role.edition));
  const expectedIds = expected.map((role) => role.id).sort();
  assert.equal(expectedIds.length, 99, 'Официальный состав базовой коробки: 99 уникальных ролей');
  assert.deepEqual(catalog.roles.map((role) => role.id).sort(), expectedIds, 'Неполный состав, дубликат или дополнительная роль');
  const ids = new Set(expectedIds);
  const counts = Object.fromEntries(['townsfolk', 'outsider', 'minion', 'demon', 'traveller', 'fabled'].map((team) => [team, catalog.roles.filter((role) => role.team === team).length]));
  assert.deepEqual(counts, { townsfolk: 39, outsider: 12, minion: 12, demon: 9, traveller: 15, fabled: 12 });
  for (const phase of phases) {
    assert.deepEqual(catalog.nightOrder[phase], reference.nightsheet[phase].filter((id) => ids.has(id) || systemSteps.has(id)), `Изменён официальный порядок ${phase}`);
  }
  for (const role of catalog.roles) {
    const source = expected.find((item) => item.id === role.id);
    for (const key of ['nameEn', 'edition', 'team', 'setup']) {
      assert.equal(role[key], source[key === 'nameEn' ? 'name' : key], `${role.id}: ${key}`);
    }
    for (const key of ['nameRu', 'abilityRu']) russianText(role[key], `${role.id}: ${key}`);
    assert.ok(Array.isArray(role.exceptionsRu) && role.exceptionsRu.length > 0, `${role.id}: нет исключений`);
    role.exceptionsRu.forEach((text) => russianText(text, `${role.id}: исключение`));
    assert.deepEqual(role.reminders.map((item) => item.key), source.reminders ?? [], `${role.id}: изменены метки или их количество`);
    role.reminders.forEach((item) => russianText(item.labelRu, `${role.id}: перевод метки`));
    for (const phase of phases) {
      const position = catalog.nightOrder[phase].indexOf(role.id);
      assert.equal(role.night[phase], position < 0 ? null : position + 1, `${role.id}: позиция ${phase}`);
    }
    assert.equal(role.source.url, `https://wiki.bloodontheclocktower.com/${source.name.replaceAll(' ', '_')}`);
    assert.equal(role.source.datasetCommit, catalog.upstream.commit);
    assert.equal(role.requirementId, `ROLE-${role.id}`);
    // Этап 1 фиксирует требования. Наличие записи не подтверждает работу движка.
    assert.deepEqual(role.implementation, { status: 'not_implemented', path: null });
    assert.equal(role.verification.behaviorStatus, 'pending');
    assert.equal(role.verification.behaviorTest, null);
    assert.equal(role.verification.catalog, 'node scripts/check-rules.mjs');
  }
  const pairs = reference.jinxes.flatMap((row) => ids.has(row.id) ? row.jinx.filter((jinx) => ids.has(jinx.id)).map((jinx) => [row.id, jinx.id]) : []);
  assert.deepEqual(catalog.baseJinxes.map((jinx) => jinx.characters), pairs, 'Неполный набор особых взаимодействий базовых ролей');
  catalog.baseJinxes.forEach((jinx) => russianText(jinx.reasonRu, `${jinx.characters.join('/')}: исключение совместимости`));
  return counts;
}

export function validateTraceability(catalog, markdown) {
  const recorded = [...markdown.matchAll(/`(ROLE-[a-z0-9]+)`/gu)].map((match) => match[1]).sort();
  assert.deepEqual(recorded, catalog.roles.map((role) => role.requirementId).sort(), 'Таблица покрытия не соответствует каталогу');
  for (const role of catalog.roles) {
    const row = markdown.split('\n').find((line) => line.includes(`\`${role.requirementId}\``));
    assert.ok(row.includes(role.source.url) && row.includes(role.abilityRu), `${role.id}: таблица потеряла источник или требование`);
    assert.ok(row.includes('Движок не реализован') && row.includes('поведенческий тест ожидается'), `${role.id}: неверный статус покрытия`);
  }
}

if (process.argv[1] && pathToFileURL(process.argv[1]).href === import.meta.url) {
  try {
    const { catalog, reference } = await loadRules();
    validateCatalog(catalog, reference);
    validateTraceability(catalog, await readFile(new URL('traceability.md', rulesRoot), 'utf8'));
    console.log('Каталог проверен: 99 ролей, 7 особых взаимодействий, состав, порядок ночи, метки и SHA-256 первоисточников. Игровые механики ещё не реализованы.');
  } catch (error) {
    console.error(`Проверка правил не пройдена: ${error.message}`);
    process.exitCode = 1;
  }
}
