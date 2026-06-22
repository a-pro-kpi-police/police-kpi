import { createContext, useContext, useState } from 'react'

const DICT = {
  ru: {
    appTitle: 'Отдел полиции',
    appSubtitle: 'Оценка эффективности сотрудников',
    login: 'Вход в систему',
    username: 'Логин',
    password: 'Пароль',
    signIn: 'Войти',
    loginError: 'Неверный логин или пароль',
    logout: 'Выйти',
    dashboard: 'Дашборд',
    entries: 'Записи KPI',
    myEntries: 'Мои записи',
    officers: 'Сотрудники',
    role_head: 'Начальник отдела',
    role_officer: 'Сотрудник',
    // дашборд
    totalFinal: 'Итоговые баллы (подтв.)',
    confirmedCount: 'Подтверждено',
    pendingCount: 'На рассмотрении',
    rejectedCount: 'Отклонено',
    officersCount: 'Сотрудников',
    confirmRate: 'Доля подтверждения',
    byOfficer: 'Эффективность по сотрудникам',
    byBlock: 'Баллы по блокам',
    trend: 'Динамика по месяцам',
    allMonths: 'Все месяцы',
    allBlocks: 'Все блоки',
    rating: 'Рейтинг эффективности',
    // колонки
    colDate: 'Дата',
    colMonth: 'Месяц',
    colOfficer: 'ФИО сотрудника',
    colCode: 'Код KPI',
    colIndicator: 'Показатель',
    colBlock: 'Блок',
    colNpa: 'НПА/статья',
    colPointsPerUnit: 'Балл за 1 ед.',
    colCoeff: 'Коэфф. участка',
    colRaw: 'Сырые баллы',
    colFinal: 'Итоговые баллы',
    colStatus: 'Статус',
    colComments: 'Комментарии',
    colActions: 'Действия',
    colQuantity: 'Кол-во',
    colRank: 'Звание',
    colDistrict: 'Участок',
    // статусы
    pending: 'На рассмотрении',
    confirmed: 'Подтверждено',
    rejected: 'Отклонено',
    // действия
    addEntry: 'Добавить запись',
    confirm: 'Подтвердить',
    reject: 'Отклонить',
    save: 'Сохранить',
    cancel: 'Отмена',
    newEntry: 'Новая запись KPI',
    selectKpi: 'Показатель KPI',
    quantity: 'Количество единиц',
    comments: 'Комментарий',
    headResolution: 'Резолюция начальника',
    computedRaw: 'Сырые баллы',
    computedFinal: 'Итоговые баллы',
    yourDistrict: 'Ваш участок',
    yourCoeff: 'Коэфф. участка',
    noEntries: 'Записей нет',
    filterStatus: 'Статус',
    filterAll: 'Все',
    periodAll: 'Все периоды',
    byMonth: 'По месяцам',
    byQuarter: 'По кварталам',
    byYear: 'По годам',
    searchName: 'Поиск по ФИО…',
    nothingFound: 'Ничего не найдено',
    confirmedFinal: 'Подтв. баллы',
    pendingShort: 'Ожидают',
    points: 'балл.',
    entryCreated: 'Запись отправлена на рассмотрение',
    entryConfirmed: 'Запись подтверждена',
    entryRejected: 'Запись отклонена',
    saveError: 'Ошибка сохранения',
    statusUpdateError: 'Не удалось обновить статус',
    validationError: 'Проверьте поля формы',
    entryDateError: 'Укажите корректную дату',
    quantityPositiveError: 'Количество должно быть больше нуля',
    catalogNotFoundError: 'Показатель KPI не найден',
    entryNotFoundError: 'Запись не найдена',
    statusInvalidError: 'Недопустимый статус',
  },
  kk: {
    appTitle: 'Полиция бөлімі',
    appSubtitle: 'Қызметкерлердің тиімділігін бағалау',
    login: 'Жүйеге кіру',
    username: 'Логин',
    password: 'Құпиясөз',
    signIn: 'Кіру',
    loginError: 'Логин немесе құпиясөз қате',
    logout: 'Шығу',
    dashboard: 'Бақылау тақтасы',
    entries: 'KPI жазбалары',
    myEntries: 'Менің жазбаларым',
    officers: 'Қызметкерлер',
    role_head: 'Бөлім бастығы',
    role_officer: 'Қызметкер',
    totalFinal: 'Қорытынды баллдар (раст.)',
    confirmedCount: 'Расталды',
    pendingCount: 'Қаралуда',
    rejectedCount: 'Қабылданбады',
    officersCount: 'Қызметкерлер',
    confirmRate: 'Растау үлесі',
    byOfficer: 'Қызметкерлер бойынша тиімділік',
    byBlock: 'Блоктар бойынша баллдар',
    trend: 'Айлар бойынша динамика',
    allMonths: 'Барлық айлар',
    allBlocks: 'Барлық блоктар',
    rating: 'Тиімділік рейтингі',
    colDate: 'Күні',
    colMonth: 'Айы',
    colOfficer: 'Қызметкердің Т.А.Ә.',
    colCode: 'KPI коды',
    colIndicator: 'Көрсеткіш',
    colBlock: 'Блок',
    colNpa: 'НҚА/бап',
    colPointsPerUnit: '1 бірлік балл',
    colCoeff: 'Учаске коэфф.',
    colRaw: 'Шикі баллдар',
    colFinal: 'Қорытынды баллдар',
    colStatus: 'Мәртебесі',
    colComments: 'Түсініктемелер',
    colActions: 'Әрекеттер',
    colQuantity: 'Саны',
    colRank: 'Атағы',
    colDistrict: 'Учаске',
    pending: 'Қаралуда',
    confirmed: 'Расталды',
    rejected: 'Қабылданбады',
    addEntry: 'Жазба қосу',
    confirm: 'Растау',
    reject: 'Қабылдамау',
    save: 'Сақтау',
    cancel: 'Болдырмау',
    newEntry: 'Жаңа KPI жазбасы',
    selectKpi: 'KPI көрсеткіші',
    quantity: 'Бірлік саны',
    comments: 'Түсініктеме',
    headResolution: 'Бастық қарары',
    computedRaw: 'Шикі баллдар',
    computedFinal: 'Қорытынды баллдар',
    yourDistrict: 'Сіздің учаске',
    yourCoeff: 'Учаске коэфф.',
    noEntries: 'Жазбалар жоқ',
    filterStatus: 'Мәртебесі',
    filterAll: 'Барлығы',
    periodAll: 'Барлық кезеңдер',
    byMonth: 'Айлар бойынша',
    byQuarter: 'Тоқсандар бойынша',
    byYear: 'Жылдар бойынша',
    searchName: 'Т.А.Ә. бойынша іздеу…',
    nothingFound: 'Ештеңе табылмады',
    confirmedFinal: 'Раст. баллдар',
    pendingShort: 'Күтуде',
    points: 'балл',
    entryCreated: 'Жазба қарауға жіберілді',
    entryConfirmed: 'Жазба расталды',
    entryRejected: 'Жазба қабылданбады',
    saveError: 'Сақтау қатесі',
    statusUpdateError: 'Мәртебені жаңарту мүмкін болмады',
    validationError: 'Форма өрістерін тексеріңіз',
    entryDateError: 'Дұрыс күнді көрсетіңіз',
    quantityPositiveError: 'Саны нөлден үлкен болуы керек',
    catalogNotFoundError: 'KPI көрсеткіші табылмады',
    entryNotFoundError: 'Жазба табылмады',
    statusInvalidError: 'Жарамсыз мәртебе',
  },
}

const MONTHS = {
  ru: ['Январь', 'Февраль', 'Март', 'Апрель', 'Май', 'Июнь',
       'Июль', 'Август', 'Сентябрь', 'Октябрь', 'Ноябрь', 'Декабрь'],
  kk: ['Қаңтар', 'Ақпан', 'Наурыз', 'Сәуір', 'Мамыр', 'Маусым',
       'Шілде', 'Тамыз', 'Қыркүйек', 'Қазан', 'Қараша', 'Желтоқсан'],
}

const I18nContext = createContext(null)

export function I18nProvider({ children }) {
  const [lang, setLang] = useState(localStorage.getItem('lang') || 'ru')
  const change = (l) => {
    setLang(l)
    localStorage.setItem('lang', l)
  }
  const t = (key) => DICT[lang][key] ?? key
  // "2026-06" -> "Июнь 2026" / "Маусым 2026"
  const fmtMonth = (ym) => {
    if (!ym || typeof ym !== 'string' || !ym.includes('-')) return ym ?? ''
    const [year, month] = ym.split('-')
    const idx = parseInt(month, 10) - 1
    const name = MONTHS[lang][idx]
    return name ? `${name} ${year}` : ym
  }
  // (year, q) -> "2 кв. 2026" / "2-тоқсан 2026"
  const fmtQuarter = (year, q) =>
    lang === 'kk' ? `${q}-тоқсан ${year}` : `${q} кв. ${year}`
  return (
    <I18nContext.Provider value={{ lang, setLang: change, t, fmtMonth, fmtQuarter }}>
      {children}
    </I18nContext.Provider>
  )
}

export const useI18n = () => useContext(I18nContext)
