import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('../api/client', () => ({
  getEvents: vi.fn().mockResolvedValue({ items: [], total: 0 }),
}))

import { getEvents } from '../api/client'
import { useEventsStore, localDayStart } from './events'

describe('events store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(getEvents).mockClear()
  })

  it('converts a local day to the UTC instant it starts at', () => {
    expect(localDayStart('2026-10-16')).toBe(new Date(2026, 9, 16).toISOString())
    expect(localDayStart('2026-10-31', 1)).toBe(new Date(2026, 10, 1).toISOString())
  })

  it('sends the "to" day as the exclusive start of the next local day', async () => {
    const store = useEventsStore()
    store.from = '2026-10-01'
    store.setFilter('to', '2026-10-02', '10.0.0.1')
    await vi.waitFor(() => expect(getEvents).toHaveBeenCalled())
    expect(vi.mocked(getEvents).mock.calls[0][0]).toMatchObject({
      page: 1,
      from: new Date(2026, 9, 1).toISOString(),
      to: new Date(2026, 9, 3).toISOString(),
      address: '10.0.0.1',
    })
  })

  it('omits empty date filters', async () => {
    const store = useEventsStore()
    await store.fetchEvents()
    const params = vi.mocked(getEvents).mock.calls[0][0]
    expect(params.from).toBeUndefined()
    expect(params.to).toBeUndefined()
  })
})
