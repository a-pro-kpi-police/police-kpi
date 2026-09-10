import test from 'node:test'
import assert from 'node:assert/strict'
import {
  blockScoreDomain,
  blockScoreLabelPosition,
  formatBlockScore,
} from './blockScoreChart.js'

test('formats positive, negative, zero, and fractional block scores', () => {
  assert.equal(formatBlockScore(12, 'балл.'), '12 балл.')
  assert.equal(formatBlockScore(-4.5, 'балл.'), '-4.5 балл.')
  assert.equal(formatBlockScore(0, 'балл.'), '0 балл.')
})

test('reserves axis space beyond positive and negative bars', () => {
  assert.deepEqual(blockScoreDomain([-10, 30]), [-20, 40])
  assert.deepEqual(blockScoreDomain([2, 10]), [0, 12.5])
  assert.deepEqual(blockScoreDomain([-10, -2]), [-12.5, 0])
  assert.deepEqual(blockScoreDomain([0, 0]), [-1, 1])
})

test('places labels at the value end of positive, negative, and zero bars', () => {
  assert.deepEqual(
    blockScoreLabelPosition({ x: 100, width: 120, value: 25, label: '25 балл.' }),
    { x: 214, textAnchor: 'end', inside: true },
  )
  assert.deepEqual(
    blockScoreLabelPosition({ x: 40, width: 120, value: -25, label: '-25 балл.' }),
    { x: 46, textAnchor: 'start', inside: true },
  )
  assert.deepEqual(
    blockScoreLabelPosition({ x: 100, width: 0, value: 0, label: '0 балл.' }),
    { x: 106, textAnchor: 'start', inside: false },
  )
})

test('moves labels outside bars that are too short', () => {
  assert.deepEqual(
    blockScoreLabelPosition({ x: 100, width: 12, value: 1, label: '1 балл.' }),
    { x: 118, textAnchor: 'start', inside: false },
  )
  assert.deepEqual(
    blockScoreLabelPosition({ x: 88, width: 12, value: -1, label: '-1 балл.' }),
    { x: 82, textAnchor: 'end', inside: false },
  )
})
