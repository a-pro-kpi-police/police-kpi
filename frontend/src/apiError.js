const DETAIL_TO_KEY = {
  'Показатель KPI не найден': 'catalogNotFoundError',
  'Количество должно быть больше нуля': 'quantityPositiveError',
  'Запись не найдена': 'entryNotFoundError',
  'Недопустимый статус': 'statusInvalidError',
}

export function getApiErrorMessage(err, t, fallbackKey = 'saveError') {
  const detail = err?.response?.data?.detail

  if (typeof detail === 'string') {
    return t(DETAIL_TO_KEY[detail] || fallbackKey)
  }

  if (Array.isArray(detail)) {
    const hasDateError = detail.some((item) => item?.loc?.includes('date'))
    if (hasDateError) return t('entryDateError')
    return t('validationError')
  }

  return t(fallbackKey)
}
