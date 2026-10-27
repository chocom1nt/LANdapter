import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';

const appSource = await readFile(new URL('../src/App.jsx', import.meta.url), 'utf8');

test('device inventory requests are gated on an active client', () => {
  assert.match(
    appSource,
    /selectedClientId\s*&&\s*selectedClientOnline\s*\?\s*getClientDevices\(selectedClientId, signal\)/,
  );
  assert.match(appSource, /devices\.clientOffline/);
});

test('live telemetry requests are gated on an active client', () => {
  assert.match(
    appSource,
    /selectedClientId\s*&&\s*selectedClientOnline\s*\?\s*getClientStats\(selectedClientId, signal\)/,
  );
  assert.match(appSource, /analytics\.agentOffline/);
});
