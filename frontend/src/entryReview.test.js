import assert from 'node:assert/strict'
import test from 'node:test'

import {
  defaultEntryStatusFilter,
  entryMatchesStatus,
  reconcileEntryStatus,
} from './entryReview.js'

test('opens the reviewer view with pending records selected', () => {
  assert.equal(defaultEntryStatusFilter(true), 'pending')
  assert.equal(defaultEntryStatusFilter(false), '')
})

test('moves a successfully updated record between status views', () => {
  const pendingEntry = { id: 9, status: 'pending' }
  const confirmedEntry = { ...pendingEntry, status: 'confirmed' }

  assert.equal(entryMatchesStatus(pendingEntry, 'pending'), true)
  assert.equal(entryMatchesStatus(confirmedEntry, 'pending'), false)
  assert.equal(entryMatchesStatus(confirmedEntry, 'confirmed'), true)
})

test('shows every record when no status section is selected', () => {
  assert.equal(entryMatchesStatus({ id: 9, status: 'rejected' }, ''), true)
})

test('removes a reviewed record from the pending page for correct pagination', () => {
  const entries = [{ id: 8, status: 'pending' }, { id: 9, status: 'pending' }]
  const updatedEntry = { id: 9, status: 'confirmed' }

  assert.deepEqual(reconcileEntryStatus(entries, updatedEntry, 'pending'), [entries[0]])
})

test('inserts a reviewed record when its destination view is already selected', () => {
  const confirmedEntry = { id: 9, status: 'confirmed' }

  assert.deepEqual(
    reconcileEntryStatus([{ id: 8, status: 'confirmed' }], confirmedEntry, 'confirmed'),
    [confirmedEntry, { id: 8, status: 'confirmed' }],
  )
})
