import assert from 'node:assert/strict'
import test from 'node:test'

import {
  effectivenessCategory,
  maxEffectivenessScore,
  relativeScorePercent,
} from './effectiveness.js'

test('uses the highest employee score as 100 percent', () => {
  const maxScore = maxEffectivenessScore([80, 100, 120])

  assert.equal(maxScore, 120)
  assert.equal(effectivenessCategory(120, maxScore), 'very_effective')
  assert.equal(effectivenessCategory(90, maxScore), 'very_effective')
  assert.equal(effectivenessCategory(89.99, maxScore), 'effective')
})

test('keeps the effective category between 50 and 75 percent of the top score', () => {
  const maxScore = 120

  assert.equal(effectivenessCategory(60, maxScore), 'effective')
  assert.equal(effectivenessCategory(59.99, maxScore), 'little_effective')
})

test('classifies zero and non-positive top scores as low effectiveness', () => {
  assert.equal(maxEffectivenessScore([0, 0, 0]), 0)
  assert.equal(effectivenessCategory(0, 0), 'little_effective')
  assert.equal(effectivenessCategory(-10, 100), 'little_effective')
})

test('supports direct percent values when max score is omitted', () => {
  assert.equal(effectivenessCategory(75), 'very_effective')
  assert.equal(effectivenessCategory(50), 'effective')
  assert.equal(effectivenessCategory(49.99), 'little_effective')
})

test('classifies employee objects by their actual score', () => {
  const officers = [
    { name: 'A', actual_score: 120 },
    { name: 'B', actual_score: 70 },
    { name: 'C', actual_score: 40 },
  ]
  const maxScore = maxEffectivenessScore(officers)

  assert.equal(effectivenessCategory(officers[0], maxScore), 'very_effective')
  assert.equal(effectivenessCategory(officers[1], maxScore), 'effective')
  assert.equal(effectivenessCategory(officers[2], maxScore), 'little_effective')
})

test('uses category score before current-month actual score', () => {
  const officers = [
    { name: 'A', category_score: 300, actual_score: 0 },
    { name: 'B', category_score: 180, actual_score: 0 },
  ]
  const maxScore = maxEffectivenessScore(officers)

  assert.equal(maxScore, 300)
  assert.equal(effectivenessCategory(officers[0], maxScore), 'very_effective')
  assert.equal(effectivenessCategory(officers[1], maxScore), 'effective')
})

test('calculates relative percent from the highest score', () => {
  assert.equal(relativeScorePercent(120, 120), 100)
  assert.equal(relativeScorePercent(60, 120), 50)
  assert.equal(relativeScorePercent(90, 120), 75)
})

test('keeps relative percent between zero and one hundred', () => {
  assert.equal(relativeScorePercent(150, 120), 100)
  assert.equal(relativeScorePercent(-10, 120), 0)
  assert.equal(relativeScorePercent(50, 0), 0)
})
