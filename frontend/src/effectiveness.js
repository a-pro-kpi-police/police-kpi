export const EFFECTIVENESS_CATEGORIES = [
  'very_effective',
  'effective',
  'little_effective',
]

const HIGH_EFFECTIVENESS_THRESHOLD = 75
const EFFECTIVE_THRESHOLD = 50

function numericScore(value) {
  const score = Number(typeof value === 'string' ? value.replace(',', '.') : value)
  return Number.isFinite(score) ? score : 0
}

function rawRelativePercent(score, maxScore) {
  const max = numericScore(maxScore)
  if (max <= 0) return 0
  const percent = (numericScore(score) / max) * 100
  return Math.min(100, Math.max(0, percent))
}

export function relativeScorePercent(score, maxScore) {
  const percent = rawRelativePercent(score, maxScore)
  return Math.round(Math.min(100, Math.max(0, percent)) * 10) / 10
}

export function maxEffectivenessScore(scores) {
  if (!scores.length) return 0
  return Math.max(...scores.map((score) => effectivenessScore(score)), 0)
}

export function effectivenessScore(value) {
  if (!value || typeof value !== 'object') return numericScore(value)
  return numericScore(
    value.category_score
      ?? value.categoryScore
      ?? value.actual_score
      ?? value.actualScore
      ?? value.final
      ?? value.score
      ?? value.achievement_percent
      ?? value.achievementPercent
      ?? 0,
  )
}

export function effectivenessCategory(score, maxScore = 100) {
  const relativePercent = rawRelativePercent(effectivenessScore(score), maxScore)

  if (relativePercent >= HIGH_EFFECTIVENESS_THRESHOLD) return 'very_effective'
  if (relativePercent >= EFFECTIVE_THRESHOLD) return 'effective'
  return 'little_effective'
}

export function compareEffectiveness(a, b) {
  const categoryDifference = (
    EFFECTIVENESS_CATEGORIES.indexOf(a.effectivenessCategory)
    - EFFECTIVENESS_CATEGORIES.indexOf(b.effectivenessCategory)
  )
  if (categoryDifference) return categoryDifference
  const scoreDifference = (
    effectivenessScore(b) - effectivenessScore(a)
  )
  return scoreDifference || (a.name || '').localeCompare(b.name || '', 'ru')
}
