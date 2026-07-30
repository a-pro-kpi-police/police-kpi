const DISTRICTS_KK = {
  'Обычный участок': 'Қалыпты учаске',
  'Повышенной сложности': 'Күрделілігі жоғары учаске',
  'Сложный участок': 'Күрделі учаске',
  'Особо сложный участок': 'Аса күрделі учаске',
}

export function districtName(district, lang) {
  const name = typeof district === 'string' ? district : district?.name
  if (!name) return ''
  return lang === 'kk' ? (DISTRICTS_KK[name] || name) : name
}
