import { expect, test } from 'vitest'

import {
  createBackendHealthChecker,
  isHealthyResponse,
} from './checkBackendHealth.ts'

test('accepts the backend health response', () => {
  expect(isHealthyResponse({ status: 'ok' })).toBe(true)
})

test('rejects malformed and unhealthy responses', () => {
  expect(isHealthyResponse({ status: 'error' })).toBe(false)
  expect(isHealthyResponse({})).toBe(false)
  expect(isHealthyResponse(null)).toBe(false)
})

test('checks the backend health endpoint and forwards the abort signal', async () => {
  const controller = new AbortController()
  let requestedUrl
  let requestedSignal

  const checkBackendHealth = createBackendHealthChecker(
    'https://api.example.test/v1',
    async (url, init) => {
      requestedUrl = url
      requestedSignal = init?.signal

      return new Response(JSON.stringify({ status: 'ok' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    },
  )
  const healthy = await checkBackendHealth(controller.signal)

  expect(healthy).toBe(true)
  expect(requestedUrl).toBe('https://api.example.test/v1/health')
  expect(requestedSignal).toBe(controller.signal)
})

test('returns false for unsuccessful and malformed responses', async () => {
  const checkUnsuccessfulBackend = createBackendHealthChecker(
    '/configured-api',
    async () => new Response(null, { status: 503 }),
  )
  const checkMalformedBackend = createBackendHealthChecker(
    '/configured-api',
    async () =>
      new Response(JSON.stringify({ status: 'starting' }), { status: 200 }),
  )
  const unsuccessful = await checkUnsuccessfulBackend()
  const malformed = await checkMalformedBackend()

  expect(unsuccessful).toBe(false)
  expect(malformed).toBe(false)
})

test('preserves network failures for the presentation boundary', async () => {
  const networkFailure = new Error('network unavailable')

  const checkBackendHealth = createBackendHealthChecker(
    '/configured-api',
    async () => {
      throw networkFailure
    },
  )

  await expect(checkBackendHealth()).rejects.toBe(networkFailure)
})
