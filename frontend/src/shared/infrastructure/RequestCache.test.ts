import { describe, expect, it, vi } from 'vitest'

import { RequestCache } from './RequestCache'

describe('RequestCache', () => {
  it('shares concurrent work while allowing one consumer to abort', async () => {
    let resolveLoad: ((value: string) => void) | undefined
    const load = vi.fn(
      () =>
        new Promise<string>((resolve) => {
          resolveLoad = resolve
        }),
    )
    const request = new RequestCache<string>()
    const controller = new AbortController()

    const first = request.run(load, controller.signal)
    const second = request.run(load)
    controller.abort()

    await expect(first).rejects.toMatchObject({ name: 'AbortError' })
    expect(load).toHaveBeenCalledTimes(1)
    if (!resolveLoad) throw new Error('load did not start')
    resolveLoad('done')
    await expect(second).resolves.toBe('done')

    const cachedLoad = vi.fn().mockResolvedValue('not used')
    await expect(request.run(cachedLoad)).resolves.toBe('done')
    expect(load).toHaveBeenCalledTimes(1)
    expect(cachedLoad).not.toHaveBeenCalled()

    request.invalidate()
    const freshLoad = vi.fn().mockResolvedValue('fresh')
    await expect(request.run(freshLoad)).resolves.toBe('fresh')
    expect(freshLoad).toHaveBeenCalledTimes(1)
  })

  it('does not reuse or cache an older request after invalidation', async () => {
    let resolveOld: ((value: string) => void) | undefined
    const cache = new RequestCache<string>()
    const oldResult = cache.run(
      () =>
        new Promise<string>((resolve) => {
          resolveOld = resolve
        }),
    )

    cache.invalidate()
    await expect(cache.run(async () => 'fresh')).resolves.toBe('fresh')
    if (!resolveOld) throw new Error('old load did not start')
    resolveOld('stale')
    await expect(oldResult).resolves.toBe('stale')

    await expect(cache.run(async () => 'unexpected')).resolves.toBe('fresh')
  })
})
