import test from 'node:test';
import assert from 'node:assert/strict';
import { validateApiBase, DEFAULT_API_BASE } from '../src/api.js';

test('default API path is relative and versioned', () => {
  assert.equal(DEFAULT_API_BASE, '/api/v1');
  assert.equal(validateApiBase('/api/v1/'), '/api/v1');
});

test('accepts local HTTP and remote HTTPS API endpoints', () => {
  assert.equal(validateApiBase('http://127.0.0.1:8080/api/v1'), 'http://127.0.0.1:8080/api/v1');
  assert.equal(validateApiBase('http://localhost:8080/api/v1'), 'http://localhost:8080/api/v1');
  assert.equal(validateApiBase('https://master.example.lan/api/v1/'), 'https://master.example.lan/api/v1');
});

test('rejects insecure remote HTTP and URLs with embedded credentials or query strings', () => {
  assert.throws(() => validateApiBase('http://192.168.1.20:8080/api/v1'), /HTTPS/);
  assert.throws(() => validateApiBase('https://admin:secret@master.example/api/v1'), /учётные данные/);
  assert.throws(() => validateApiBase('https://master.example/api/v1?token=secret'), /query-параметров/);
  assert.throws(() => validateApiBase('https://master.example/api'), /api\/v1/);
});
