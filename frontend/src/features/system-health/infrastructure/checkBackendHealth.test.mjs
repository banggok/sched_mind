import assert from 'node:assert/strict'
import test from 'node:test'

import {
  checkBackendHealth,
  isHealthyResponse,
} from './checkBackendHealth.ts'

test('accepts the backend health response', () => {
  assert.equal(isHealthyResponse({ status: 'ok' }), true)
})

test('rejects malformed and unhealthy responses', () => {
  assert.equal(isHealthyResponse({ status: 'error' }), false)
  assert.equal(isHealthyResponse({}), false)
  assert.equal(isHealthyResponse(null), false)
})

test('checks the backend health endpoint and forwards the abort signal', async () => {
  const controller = new AbortController()
  let requestedUrl
  let requestedSignal

  const healthy = await checkBackendHealth(
    controller.signal,
    async (url, init) => {
      requestedUrl = url
      requestedSignal = init?.signal

      return new Response(JSON.stringify({ status: 'ok' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    },
  )

  assert.equal(healthy, true)
  assert.equal(requestedUrl, '/api/health')
  assert.equal(requestedSignal, controller.signal)
})

test('returns false for unsuccessful and malformed responses', async () => {
  const unsuccessful = await checkBackendHealth(
    undefined,
    async () => new Response(null, { status: 503 }),
  )
  const malformed = await checkBackendHealth(
    undefined,
    async () =>
      new Response(JSON.stringify({ status: 'starting' }), { status: 200 }),
  )

  assert.equal(unsuccessful, false)
  assert.equal(malformed, false)
})

test('preserves network failures for the presentation boundary', async () => {
  const networkFailure = new Error('network unavailable')

  await assert.rejects(
    checkBackendHealth(undefined, async () => {
      throw networkFailure
    }),
    networkFailure,
  )
})
