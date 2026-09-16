import assert from 'node:assert/strict'
import test from 'node:test'

import {
  averageEffectivenessScore,
  effectivenessCategory,
  relativeScorePercent,
} from './effectiveness.js'

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

test('recalculates from the selected district and period, including inactive officers', () => {
  const july = [
    { district_id: 1, final: 100 }, { district_id: 1, final: 0 },
    { district_id: 2, final: 500 },
  ]
  const august = [{ district_id: 1, final: 200 }, { district_id: 1, final: 200 }]
  const districtJuly = july.filter((row) => row.district_id === 1)
  assert.equal(relativeScorePercent(100, averageEffectivenessScore(july)), 25)
  assert.equal(relativeScorePercent(100, averageEffectivenessScore(districtJuly)), 100)
  assert.equal(relativeScorePercent(200, averageEffectivenessScore(august)), 50)
})

test('uses the formula for a nonzero negative average and keeps display nonnegative', () => {
  const average = averageEffectivenessScore([-100, -100])
  assert.equal(relativeScorePercent(-100, average), 50)
  assert.equal(effectivenessCategory(-100, average), 'effective')
  assert.equal(relativeScorePercent(-10, 100), 0)
  assert.equal(effectivenessCategory(-10, 100), 'little_effective')
})
