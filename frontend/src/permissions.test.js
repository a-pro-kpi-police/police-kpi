import test from 'node:test'
import assert from 'node:assert/strict'
import { rolePermissions } from './permissions.js'

test('leadership can view team KPI without administrative or business write access', () => {
  const permissions = rolePermissions('leadership')
  assert.equal(permissions.canViewOverview, true)
  assert.equal(permissions.canViewEntries, true)
  assert.equal(permissions.canManageEntries, false)
  assert.equal(permissions.isAdmin, false)
  assert.equal(permissions.canUseCommunication, false)
})

test('existing roles retain viewing, moderation and communication permissions', () => {
  for (const [role, overview, entries, manage, communication] of [
    ['officer', false, false, false, true],
    ['head', true, true, true, true],
    ['admin', false, true, true, false],
  ]) {
    const permissions = rolePermissions(role)
    assert.equal(permissions.canViewOverview, overview, role)
    assert.equal(permissions.canViewEntries, entries, role)
    assert.equal(permissions.canManageEntries, manage, role)
    assert.equal(permissions.canUseCommunication, communication, role)
  }
})
