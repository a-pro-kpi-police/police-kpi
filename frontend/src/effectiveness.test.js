import assert from 'node:assert/strict'
import test from 'node:test'

import {
  averageEffectivenessScore,
  effectivenessCategory,
} from './effectiveness.js'

test('uses the employee average as the start of middle effectiveness', () => {
  const average = averageEffectivenessScore([80, 100, 120])

  assert.equal(average, 100)
  assert.equal(effectivenessCategory(99.99, average), 'little_effective')
  assert.equal(effectivenessCategory(100, average), 'middle_effective')
})

test('includes employees with zero achievement in the average', () => {
  assert.equal(averageEffectivenessScore([0, 60, 90]), 50)
})

test('keeps effective and very effective thresholds weighted above the average', () => {
  const average = 100

  assert.equal(effectivenessCategory(129.99, average), 'middle_effective')
  assert.equal(effectivenessCategory(130, average), 'effective')
  assert.equal(effectivenessCategory(169.99, average), 'effective')
  assert.equal(effectivenessCategory(170, average), 'very_effective')
})

test('handles a zero employee average without classifying zero as very effective', () => {
  assert.equal(effectivenessCategory(0, 0), 'middle_effective')
  assert.equal(effectivenessCategory(1, 0), 'very_effective')
  assert.equal(effectivenessCategory(-1, 0), 'little_effective')
})
