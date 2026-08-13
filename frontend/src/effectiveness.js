export const EFFECTIVENESS_CATEGORIES = [
  'very_effective',
  'effective',
  'middle_effective',
  'little_effective',
]

export const EFFECTIVENESS_WEIGHTS = {
  very_effective: 1.7,
  effective: 1.3,
  middle_effective: 1,
}

function numericScore(value) {
  const score = Number(value)
  return Number.isFinite(score) ? score : 0
}

export function averageEffectivenessScore(scores) {
  if (!scores.length) return 0
  return scores.reduce((sum, score) => sum + numericScore(score), 0) / scores.length
}

export function effectivenessCategory(achievementPercent, averageScore = 60) {
  const value = numericScore(achievementPercent)
  const average = numericScore(averageScore)

  // When the average is negative, use its magnitude for the category spacing
  // so the thresholds still rise from the exact employee average.
  const averageMagnitude = Math.abs(average)
  if (averageMagnitude === 0) {
    if (value > average) return 'very_effective'
    if (value === average) return 'middle_effective'
    return 'little_effective'
  }

  const weightedThreshold = (weight) => (
    average + averageMagnitude * (weight - EFFECTIVENESS_WEIGHTS.middle_effective)
  )
  if (value >= weightedThreshold(EFFECTIVENESS_WEIGHTS.very_effective)) {
    return 'very_effective'
  }
  if (value >= weightedThreshold(EFFECTIVENESS_WEIGHTS.effective)) return 'effective'
  if (value >= average) return 'middle_effective'
  return 'little_effective'
}

export function compareEffectiveness(a, b) {
  const categoryDifference = (
    EFFECTIVENESS_CATEGORIES.indexOf(a.effectivenessCategory)
    - EFFECTIVENESS_CATEGORIES.indexOf(b.effectivenessCategory)
  )
  if (categoryDifference) return categoryDifference
  const achievementDifference = (
    Number(b.achievement_percent || 0) - Number(a.achievement_percent || 0)
  )
  return achievementDifference || (a.name || '').localeCompare(b.name || '', 'ru')
}
