const DETAIL_TO_KEY = {
  'Выберите хотя бы одну роту': 'selectAtLeastOneCompany',
  'Укажите доступ к ротам': 'companyAccessRequiredError',
  'Рота указана неверно': 'companyInvalidError',
  'Показатель KPI не найден': 'catalogNotFoundError',
  'Код KPI обязателен': 'kpiCodeRequiredError',
  'Показатель KPI обязателен': 'kpiIndicatorRequiredError',
  'Блок KPI обязателен': 'kpiBlockRequiredError',
  'Код KPI уже занят': 'kpiCodeTakenError',
  'Балл KPI должен быть нулём или больше': 'kpiPointsNonNegativeError',
  'Лимит KPI не может быть отрицательным': 'kpiLimitNonNegativeError',
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
  'Слишком много вложений': 'attachmentCountError',
  'Вложение не найдено': 'attachmentNotFoundError',
  'Логин обязателен': 'usernameRequiredError',
  'Пароль обязателен': 'passwordRequiredError',
  'ФИО сотрудника обязательно': 'fullNameRequiredError',
  'Укажите участок сотрудника': 'districtRequiredError',
  'Участок не найден': 'districtNotFoundError',
  'Логин уже занят': 'usernameTakenError',
  'Табельный номер уже занят': 'employeeNumberTakenError',
  'Месячная норма должна быть больше нуля': 'monthlyNormPositiveError',
  'Недопустимая роль пользователя': 'userRoleInvalidError',
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
