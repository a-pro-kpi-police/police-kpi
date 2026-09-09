import test from 'node:test'
import assert from 'node:assert/strict'

import {
  adminOfficerDashboardEndpoints,
  adminOfficerDashboardPath,
  openAdminOfficerDashboard,
  openAdminRatingOfficerDashboard,
} from './adminOfficerDashboard.js'

test('officer name click navigates to the protected admin dashboard route', () => {
  const visited = []
  const employee = { id: 42, role: 'officer', full_name: 'Иванов Иван' }

  const path = openAdminOfficerDashboard(employee, (value) => visited.push(value))

  assert.equal(path, '/admin/officers/42')
  assert.deepEqual(visited, ['/admin/officers/42'])
  assert.deepEqual(adminOfficerDashboardEndpoints('42'), {
    dashboard: '/admin/officers/42/dashboard',
    targetOverview: '/admin/officers/42/target-overview',
  })
})

test('non-officers and invalid ids do not create dashboard navigation', () => {
  const visited = []

  assert.equal(adminOfficerDashboardPath({ id: 7, role: 'head' }), null)
  assert.equal(openAdminOfficerDashboard({ id: 0, role: 'officer' }, (value) => visited.push(value)), null)
  assert.equal(adminOfficerDashboardEndpoints('abc'), null)
  assert.deepEqual(visited, [])
})

test('officer name in admin rating opens the same personal dashboard', () => {
  const visited = []

  const path = openAdminRatingOfficerDashboard(
    { officer_id: 17, name: 'Рейтинговый Сотрудник' },
    (value) => visited.push(value),
  )

  assert.equal(path, '/admin/officers/17')
  assert.deepEqual(visited, ['/admin/officers/17'])
})
