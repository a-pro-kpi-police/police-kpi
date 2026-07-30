const DETAIL_TO_KEY = {
  'Показатель KPI не найден': 'catalogNotFoundError',
  'Количество должно быть больше нуля': 'quantityPositiveError',
  'Месяц указан неверно': 'monthInvalidError',
  'Сотрудник не указан': 'officerRequiredError',
  'Сотрудник не найден': 'officerNotFoundError',
  'Группа не найдена': 'groupNotFoundError',
  'В группе нет сотрудников': 'groupEmptyError',
  'KPI с отрицательным баллом доступен только начальнику отдела': 'negativeKpiManagerOnlyError',
  'Запись не найдена': 'entryNotFoundError',
  'Недопустимый статус': 'statusInvalidError',
  'Файл не прикреплён': 'attachmentMissingError',
  'Недопустимый тип файла': 'attachmentTypeError',
  'Файл слишком большой': 'attachmentTooLargeError',
  'Вложение не найдено': 'attachmentNotFoundError',
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
