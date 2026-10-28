import test from 'node:test';
import assert from 'node:assert/strict';
import { supportedLanguages, translations, defaultLanguage } from '../src/i18n/translations.js';

test('every supported language has the same translation keys', () => {
  const baseKeys = Object.keys(translations[defaultLanguage]).sort();
  for (const { code } of supportedLanguages) {
    assert.ok(translations[code], `missing dictionary for ${code}`);
    assert.deepEqual(Object.keys(translations[code]).sort(), baseKeys, `key mismatch for ${code}`);
  }
});

test('translations are non-empty strings and interpolation placeholders match', () => {
  for (const key of Object.keys(translations[defaultLanguage])) {
    const base = translations[defaultLanguage][key];
    assert.equal(typeof base, 'string', `${key} should be a string`);
    assert.ok(base.trim(), `${key} should not be empty`);
    const placeholders = (value) => [...value.matchAll(/\{([a-zA-Z0-9_]+)\}/g)].map((m) => m[1]).sort();
    for (const { code } of supportedLanguages) {
      assert.deepEqual(placeholders(translations[code][key]), placeholders(base), `${code}.${key} placeholders mismatch`);
    }
  }
});

test('every translation key referenced by the main UI exists', async () => {
  const { readFile } = await import('node:fs/promises');
  const appSource = await readFile(new URL('../src/App.jsx', import.meta.url), 'utf8');
  const referenced = [...appSource.matchAll(/t\(['"]([^'"]+)['"]/g)].map((match) => match[1]);
  const available = new Set(Object.keys(translations[defaultLanguage]));
  const missing = [...new Set(referenced)].filter((key) => !available.has(key));
  assert.deepEqual(missing, [], `UI references missing translations: ${missing.join(', ')}`);
});


test('sidebar navigation renders translated labels from navigation keys', async () => {
  const { readFile } = await import('node:fs/promises');
  const appSource = await readFile(new URL('../src/App.jsx', import.meta.url), 'utf8');
  assert.match(appSource, /NAVIGATION\.map\(\(\{\s*to,\s*key,\s*icon:\s*Icon,\s*end\s*\}\)\s*=>/);
  assert.match(appSource, /<span>\{t\(key\)\}<\/span>/);
  const navStart = appSource.indexOf('{NAVIGATION.map');
  const navEnd = appSource.indexOf('</nav>', navStart);
  assert.ok(navStart >= 0 && navEnd > navStart, 'sidebar navigation block should exist');
  assert.doesNotMatch(appSource.slice(navStart, navEnd), /<span>\{label\}<\/span>/);
});
