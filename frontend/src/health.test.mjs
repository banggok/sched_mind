import assert from 'node:assert/strict'
import test from 'node:test'

import { isHealthyResponse } from './health.ts'

test('accepts the backend health response', () => {
  assert.equal(isHealthyResponse({ status: 'ok' }), true)
})

test('rejects malformed and unhealthy responses', () => {
  assert.equal(isHealthyResponse({ status: 'error' }), false)
  assert.equal(isHealthyResponse({}), false)
  assert.equal(isHealthyResponse(null), false)
})

