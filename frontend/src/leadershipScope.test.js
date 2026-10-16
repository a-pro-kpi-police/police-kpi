import test from 'node:test'
import assert from 'node:assert/strict'
import { leadershipScopeForm, validLeadershipScope, leadershipScopePayload } from './leadershipScope.js'

test('leadership defaults to selected with no automatic affiliation grant', () => {
  const scope = leadershipScopeForm({ district: { id: 1 } })
  assert.deepEqual(scope, { mode: 'selected', district_ids: [] })
  assert.equal(validLeadershipScope('leadership', scope), false)
  for (const role of ['officer', 'head', 'admin']) assert.equal(validLeadershipScope(role, scope), true)
})

test('one, multiple and all companies can be restored and saved', () => {
  for (const ids of [[1], [1, 3]]) {
    const original = { mode: 'selected', district_ids: ids }
    const scope = leadershipScopeForm({ leadership_scope: original })
    assert.equal(validLeadershipScope('leadership', scope), true)
    assert.deepEqual(leadershipScopePayload('leadership', scope), original)
    assert.notEqual(scope.district_ids, original.district_ids)
  }
  const scope = leadershipScopeForm({ leadership_scope: { mode: 'all', district_ids: [1] } })
  assert.equal(validLeadershipScope('leadership', scope), true)
  assert.deepEqual(leadershipScopePayload('leadership', scope), { mode: 'all', district_ids: [] })
  assert.equal(leadershipScopePayload('head', scope), undefined)
})
