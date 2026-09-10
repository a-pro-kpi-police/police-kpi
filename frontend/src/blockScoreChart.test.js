import test from 'node:test'
import assert from 'node:assert/strict'
import {
  blockScoreDomain,
  blockScoreLabelPosition,
  blockScoreRows,
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

test('places labels outside at the value end of positive, negative, and zero bars', () => {
  assert.deepEqual(
    blockScoreLabelPosition({ x: 100, width: 120, value: 25 }),
    { x: 226, textAnchor: 'start' },
  )
  assert.deepEqual(
    blockScoreLabelPosition({ x: 40, width: 120, value: -25 }),
    { x: 34, textAnchor: 'end' },
  )
  assert.deepEqual(
    blockScoreLabelPosition({ x: 100, width: 0, value: 0 }),
    { x: 106, textAnchor: 'start' },
  )
})

test('uses the selected officer block scores and updates them between officers', () => {
  const dashboard = {
    blocks: [
      { block: 'Профилактика', final: 99 },
      { block: 'Раскрытие', final: 88 },
      { block: 'Качество', final: 77 },
    ],
    officers: [
      {
        officer_id: 7,
        blocks: { Профилактика: 12.5, Раскрытие: -4 },
      },
      {
        officer_id: 8,
        blocks: { Профилактика: 3, Раскрытие: 6.75, Качество: 0 },
      },
    ],
  }

  assert.deepEqual(blockScoreRows(dashboard, '7'), [
    { block: 'Профилактика', final: 12.5 },
    { block: 'Раскрытие', final: -4 },
    { block: 'Качество', final: 0 },
  ])
  assert.deepEqual(blockScoreRows(dashboard, '8'), [
    { block: 'Профилактика', final: 3 },
    { block: 'Раскрытие', final: 6.75 },
    { block: 'Качество', final: 0 },
  ])
})

test('keeps summary rows on the main dashboard and hides stale personal rows', () => {
  const dashboard = {
    blocks: [{ block: 'Профилактика', final: 21 }],
    officers: [{ officer_id: 7, blocks: { Профилактика: 12 } }],
  }

  assert.deepEqual(blockScoreRows(dashboard, ''), dashboard.blocks)
  assert.deepEqual(blockScoreRows(dashboard, '8'), [])
})
