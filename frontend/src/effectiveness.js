export const EFFECTIVENESS_CATEGORIES = [
  'very_effective',
  'effective',
  'middle_effective',
  'little_effective',
]

export function effectivenessCategory(achievementPercent) {
  const value = Number(achievementPercent) || 0
  if (value >= 90) return 'very_effective'
  if (value >= 75) return 'effective'
  if (value >= 60) return 'middle_effective'
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
