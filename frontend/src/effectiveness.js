export const EFFECTIVENESS_CATEGORIES = [
  'very_effective',
  'effective',
  'little_effective',
]

const HIGH_EFFECTIVENESS_THRESHOLD = 66
const EFFECTIVE_THRESHOLD = 33
const ADMIN_HIGH_PERCENT_MIN = 66.6

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

// Determine categories first; the display scale must not change group membership.
export function adminRatingEffectiveness(officers) {
  const rows = officers.map((officer) => withRelativeEffectiveness(officer))
  const highScores = rows
    .filter((officer) => officer.effectivenessCategory === 'very_effective')
    .map(effectivenessScore)
  if (!highScores.length) return rows

  const minScore = Math.min(...highScores)
  const maxScore = Math.max(...highScores)
  return rows.map((officer) => {
    if (officer.effectivenessCategory !== 'very_effective') return officer
    const percent = maxScore === minScore
      ? 100
      : ADMIN_HIGH_PERCENT_MIN
        + ((effectivenessScore(officer) - minScore) / (maxScore - minScore))
          * (100 - ADMIN_HIGH_PERCENT_MIN)
    return { ...officer, achievement_percent: Math.round(percent * 10) / 10 }
  })
}

export function dashboardOfficerEffectiveness(officer, monthlyTarget, averageScore) {
  const score = numericScore(officer.final)
  return withRelativeEffectiveness({
    ...monthlyTarget,
    ...officer,
    actual_score: score,
    gap: monthlyTarget ? Math.round((score - numericScore(monthlyTarget.target_score)) * 100) / 100 : undefined,
  }, averageScore)
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
