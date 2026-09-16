import test from 'node:test'
import assert from 'node:assert/strict'
import { loadMonthlyTargetOverview, selectedMonthOverview } from './dashboardPeriod.js'

test('all-month dashboards never fetch a latest-month target summary', async () => {
  for (const endpoint of ['/target-overview', '/admin/officers/42/target-overview']) {
    const api = { get() { assert.fail('monthly targets must not be requested') } }
    assert.deepEqual(await loadMonthlyTargetOverview(api, endpoint, { month: '', officer_id: 42 }), { data: null })
  }
})

test('selected month keeps the endpoint and employee/district scope', async () => {
  const overview = { month: '2026-07', target_score: 100 }
  const params = { month: '2026-07', officer_id: 42, district_id: 2 }
  const api = { async get(endpoint, options) {
    assert.equal(endpoint, '/target-overview')
    assert.deepEqual(options.params, params)
    return { data: overview }
  } }
  const response = await loadMonthlyTargetOverview(api, '/target-overview', params)
  assert.equal(selectedMonthOverview(response.data, params.month), overview)
})

test('switching to all months or another month hides stale monthly summaries immediately', () => {
  const overview = { month: '2026-07', target_score: 100 }
  assert.equal(selectedMonthOverview(overview, ''), null)
  assert.equal(selectedMonthOverview(overview, '2026-08'), null)
  assert.equal(selectedMonthOverview(null, '2026-07'), null)
  assert.equal(selectedMonthOverview(overview, '2026-07'), overview)
})
