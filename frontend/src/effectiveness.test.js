import assert from 'node:assert/strict'
import test from 'node:test'

import {
  averageEffectivenessScore,
  withRelativeEffectiveness,
  dashboardOfficerEffectiveness,
  effectivenessCategory,
  relativeScorePercent,
} from './effectiveness.js'

test('dashboard fact, gap and percentage use the same KPI including penalties', () => {
  const row = dashboardOfficerEffectiveness(
    { officer_id: 1, final: 80 },
    { actual_score: 100, target_score: 100, gap: 0, penalty_score: -20 },
    70,
  )
  assert.equal(row.actual_score, 80)
  assert.equal(row.gap, -20)
  assert.equal(row.achievement_percent, 57.1)
  assert.equal(row.effectivenessCategory, 'effective')
  assert.equal(row.penalty_score, -20)
  const allMonths = dashboardOfficerEffectiveness({ final: 160 }, undefined, 100)
  assert.equal(allMonths.achievement_percent, 80)
  assert.equal(allMonths.gap, undefined)
  assert.equal(allMonths.target_score, undefined)
})

test('maps the average score to 50 percent, including zero-score employees', () => {
  const average = averageEffectivenessScore([0, 100, 200])
  assert.equal(average, 100)
  assert.equal(relativeScorePercent(100, average), 50)
  assert.equal(effectivenessCategory(100, average), 'effective')
})

test('matches the issue examples and caps display without modifying scores', () => {
  const cases = [
    [50, 25, 'little_effective'],
    [66, 33, 'little_effective'],
    [100, 50, 'effective'],
    [132, 66, 'effective'],
    [150, 75, 'very_effective'],
    [250, 100, 'very_effective'],
  ]
  for (const [score, percent, category] of cases) {
    const officer = Object.freeze({ final: score })
    assert.equal(relativeScorePercent(officer.final, 100), percent)
    assert.equal(effectivenessCategory(officer, 100), category)
    assert.equal(officer.final, score)
  }
})

test('classifies exact boundaries and nearby values before display rounding', () => {
  for (const [score, category] of [
    [65.99, 'little_effective'], [66, 'little_effective'], [66.01, 'effective'],
    [131.99, 'effective'], [132, 'effective'], [132.01, 'very_effective'],
  ]) {
    assert.equal(effectivenessCategory(score, 100), category)
  }
  assert.equal(relativeScorePercent(66.01, 100), 33)
  assert.equal(relativeScorePercent(132.01, 100), 66)
})

test('handles empty ratings, zero averages and invalid scores without NaN', () => {
  for (const scores of [[], [0, 0], [-10, 10], [null, undefined, 'invalid', Infinity]]) {
    const average = averageEffectivenessScore(scores)
    assert.equal(average, 0)
    assert.equal(relativeScorePercent(100, average), 0)
    assert.equal(effectivenessCategory(100, average), 'little_effective')
  }
})

test('preserves direct-percent callers such as block progress', () => {
  assert.equal(effectivenessCategory(33), 'little_effective')
  assert.equal(effectivenessCategory(33.01), 'effective')
  assert.equal(effectivenessCategory(66), 'effective')
  assert.equal(effectivenessCategory(66.01), 'very_effective')
})

test('uses period category scores and supports API score shapes', () => {
  const officers = [
    { category_score: 100, actual_score: 0 },
    { actual_score: 200 },
    { final: 0 },
  ]
  assert.equal(averageEffectivenessScore(officers), 100)
  assert.equal(effectivenessCategory(officers[0], 100), 'effective')
  assert.equal(averageEffectivenessScore(['10,5', '20.5']), 15.5)
})

test('uses the server average regardless of district, employee, name or category filters', () => {
  const rows = [
    ...Array.from({ length: 10 }, (_, id) => ({ officer_id: id, name: 'First ' + id, district_id: 1, final: 100, global_average_score: 70 })),
    ...Array.from({ length: 30 }, (_, id) => ({ officer_id: id + 10, name: 'Second ' + id, district_id: 2, final: 60, global_average_score: 70 })),
  ]
  for (const filter of [() => true, row => row.district_id === 1, row => row.officer_id === 0, row => row.name === 'First 0']) {
    const visible = rows.filter(filter).map(row => withRelativeEffectiveness(row))
    assert.equal(visible[0].achievement_percent, 71.4)
    assert.equal(visible[0].effectivenessCategory, 'very_effective')
    assert.equal(visible.filter(row => row.effectivenessCategory === 'very_effective')[0].achievement_percent, 71.4)
  }
  assert.deepEqual(rows.filter(() => false).map(row => withRelativeEffectiveness(row)), [])
  assert.equal(withRelativeEffectiveness({ final: 0, global_average_score: 0 }).achievement_percent, 0)
  assert.equal(withRelativeEffectiveness({ final: 200, global_average_score: 5 }).achievement_percent, 100)
  // Dashboard all-month scores and average must describe the same period.
  assert.equal(withRelativeEffectiveness({ final: 300 }, 75).achievement_percent, 100)
})

test('uses the formula for a nonzero negative average and keeps display nonnegative', () => {
  const average = averageEffectivenessScore([-100, -100])
  assert.equal(relativeScorePercent(-100, average), 50)
  assert.equal(effectivenessCategory(-100, average), 'effective')
  assert.equal(relativeScorePercent(-10, 100), 0)
  assert.equal(effectivenessCategory(-10, 100), 'little_effective')
})

test('weights every employee equally across companies of different sizes (#20)', () => {
  for (const scoreKey of ['final', 'actual_score', 'category_score']) {
    const officers = [
      ...Array.from({ length: 10 }, () => ({ district_id: 1, [scoreKey]: 100 })),
      ...Array.from({ length: 30 }, () => ({ district_id: 2, [scoreKey]: 60 })),
    ]
    assert.equal(averageEffectivenessScore(officers), 70)
    assert.equal(averageEffectivenessScore(officers.filter((row) => row.district_id === 1)), 100)
    assert.equal(averageEffectivenessScore(officers.filter((row) => row.district_id === 2)), 60)
    assert.equal(averageEffectivenessScore([...officers, { district_id: 3, [scoreKey]: 0 }]), 2800 / 41)
  }
})
