import assert from 'node:assert/strict'
import test from 'node:test'

import { mergeSupportMessages, parseApiTimestamp } from './supportMessages.js'

test('parses API timestamps without timezone as UTC', () => {
  assert.equal(
    parseApiTimestamp('2026-09-01T12:00:00').toISOString(),
    '2026-09-01T12:00:00.000Z',
  )
})

test('keeps locally appended support messages when a stale poll resolves', () => {
  const current = [
    { id: 1, body: 'before send', created_at: '2026-09-01T12:00:00Z' },
    { id: 2, body: 'sent message', created_at: '2026-09-01T12:01:00Z' },
  ]
  const stalePoll = [
    { id: 1, body: 'before send', created_at: '2026-09-01T12:00:00Z' },
  ]

  assert.deepEqual(
    mergeSupportMessages(current, stalePoll).map((message) => message.id),
    [1, 2],
  )
})

test('deduplicates support messages returned by both poll and send', () => {
  const poll = [
    { id: 1, body: 'before send', created_at: '2026-09-01T12:00:00Z' },
    { id: 2, body: 'sent from poll', created_at: '2026-09-01T12:01:00Z' },
  ]
  const sendResponse = [
    { id: 2, body: 'sent from post', created_at: '2026-09-01T12:01:00Z' },
  ]

  const merged = mergeSupportMessages(poll, sendResponse)

  assert.deepEqual(merged.map((message) => message.id), [1, 2])
  assert.equal(merged[1].body, 'sent from post')
})
