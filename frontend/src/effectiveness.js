export const EFFECTIVENESS_CATEGORIES = [
  'very_effective',
  'effective',
  'little_effective',
]

const HIGH_EFFECTIVENESS_THRESHOLD = 66
const EFFECTIVE_THRESHOLD = 33

function numericScore(value) {
  const score = Number(typeof value === 'string' ? value.replace(',', '.') : value)
  return Number.isFinite(score) ? score : 0
}

function rawRelativePercent(score, averageScore) {
  const average = numericScore(averageScore)
  if (average === 0) return 0
  return (numericScore(score) / average) * 50
}

export function relativeScorePercent(score, averageScore) {
  const percent = rawRelativePercent(score, averageScore)
  return Math.round(Math.min(100, Math.max(0, percent)) * 10) / 10
}

// The server aggregate includes employees outside the visible list.
export function withRelativeEffectiveness(officer, averageScore = officer.global_average_score) {
  return {
    ...officer,
    achievement_percent: relativeScorePercent(effectivenessScore(officer), averageScore),
    effectivenessCategory: effectivenessCategory(officer, averageScore),
  }
}

export function averageEffectivenessScore(scores) {
  if (!scores.length) return 0
  return scores.reduce((sum, score) => sum + effectivenessScore(score), 0) / scores.length
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

export function effectivenessCategory(score, averageScore = 50) {
  const relativePercent = rawRelativePercent(effectivenessScore(score), averageScore)

  if (relativePercent > HIGH_EFFECTIVENESS_THRESHOLD) return 'very_effective'
  if (relativePercent > EFFECTIVE_THRESHOLD) return 'effective'
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
