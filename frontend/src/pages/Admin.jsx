import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '../api'
import { getApiErrorMessage } from '../apiError.js'
import NameFilterCombobox from '../components/NameFilterCombobox.jsx'
import { districtName } from '../districtLocale.js'
import {
  EFFECTIVENESS_CATEGORIES,
  effectivenessCategory,
  maxEffectivenessScore,
  relativeScorePercent,
} from '../effectiveness.js'
import { useI18n } from '../i18n.jsx'
import { useNotify } from '../notifications.jsx'
import AdminSupportChat from './AdminSupportChat.jsx'

const RANKS = [
  'Сержант полиции',
  'Старший сержант полиции',
  'Лейтенант полиции',
  'Старший лейтенант полиции',
  'Капитан полиции',
  'Майор полиции',
  'Подполковник полиции',
  'Полковник полиции',
]

const rankOrder = (rank) => {
  const index = RANKS.indexOf(rank)
  return index === -1 ? RANKS.length : index
}

const compareRatingNames = (a, b) => (a.name || '').localeCompare(b.name || '', 'ru')

function SortTh({ column, label, sort, onSort }) {
  const active = sort.key === column
  return (
    <th
      className={`sortable${active ? ' sorted' : ''}`}
      aria-sort={active ? (sort.dir === 'asc' ? 'ascending' : 'descending') : 'none'}
      onClick={() => onSort(column)}
    >
      {label}
      <span className="sort-ind">{active ? (sort.dir === 'asc' ? '▲' : '▼') : '⇅'}</span>
    </th>
  )
}

function EffectivenessBadge({ category, t }) {
  const labelKey = {
    very_effective: 'categoryVeryEffective',
    effective: 'categoryEffective',
    little_effective: 'categoryLittleEffective',
  }[category]
  return (
    <span className={`effectiveness-badge effectiveness-${category}`}>
      {t(labelKey)}
    </span>
  )
}

const newEmployeeForm = (districts) => ({
  username: '',
  password: 'pass123',
  full_name: '',
  rank: '',
  role: 'officer',
  district_id: districts[0]?.id ? String(districts[0].id) : '',
  source_employee_number: '',
  monthly_norm: 100,
})

const employeeForm = (employee, districts) => {
  if (!employee) return newEmployeeForm(districts)
  return {
    username: employee.username || '',
    password: '',
    full_name: employee.full_name || '',
    rank: employee.rank || '',
    role: employee.role || 'officer',
    district_id: employee.district?.id ? String(employee.district.id) : '',
    source_employee_number: employee.source_employee_number || '',
    monthly_norm: employee.monthly_norm || 100,
  }
}

const newCatalogForm = () => ({
  code: '',
  indicator: '',
  block: '',
  subblock: '',
  npa: '',
  points_per_unit: 1,
  unit: 'материал',
  source_url: '',
  monthly_limit: 0,
  is_mass: false,
  is_critical: false,
})

const catalogForm = (item) => {
  if (!item) return newCatalogForm()
  return {
    code: item.code || '',
    indicator: item.indicator || '',
    block: item.block || '',
    subblock: item.subblock || '',
    npa: item.npa || '',
    points_per_unit: item.points_per_unit ?? 1,
    unit: item.unit || 'материал',
    source_url: item.source_url || '',
    monthly_limit: item.monthly_limit ?? 0,
    is_mass: Boolean(item.is_mass),
    is_critical: Boolean(item.is_critical),
  }
}

function formatTemplate(template, values) {
  return Object.entries(values).reduce(
    (text, [key, value]) => text.replaceAll(`{${key}}`, String(value)),
    template,
  )
}

function EmployeeModal({ employee, districts, onClose, onSaved }) {
  const { t, lang } = useI18n()
  const { notify } = useNotify()
  const [form, setForm] = useState(() => employeeForm(employee, districts))
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    setForm((current) => (
      current.district_id || !districts[0]?.id
        ? current
        : { ...current, district_id: String(districts[0].id) }
    ))
  }, [districts])

  const updateField = (field, value) => {
    setForm((current) => ({ ...current, [field]: value }))
  }

  const submit = async (event) => {
    event.preventDefault()
    setSaving(true)
    setError('')
    try {
      const payload = {
        username: form.username.trim(),
        password: form.password.trim(),
        full_name: form.full_name.trim(),
        rank: form.rank.trim(),
        role: form.role,
        district_id: form.district_id ? Number(form.district_id) : null,
        source_employee_number: form.source_employee_number.trim(),
        monthly_norm: Number(form.monthly_norm),
      }
      const response = employee
        ? await api.patch(`/admin/users/${employee.id}`, payload)
        : await api.post('/admin/users', payload)
      onSaved(response.data, employee ? 'updated' : 'created')
      notify(t(employee ? 'employeeSaved' : 'employeeCreated'))
      onClose()
    } catch (err) {
      const message = getApiErrorMessage(err, t, 'saveError')
      setError(message)
      notify(message, 'error')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal employee-form-modal" onClick={(event) => event.stopPropagation()}>
        <h2>{t(employee ? 'editEmployee' : 'addEmployee')}</h2>
        <form className="employee-form" onSubmit={submit}>
          <div className="form-row">
            <div>
              <label>{t('username')}</label>
              <input
                value={form.username}
                onChange={(event) => updateField('username', event.target.value)}
                required
              />
            </div>
            <div>
              <label>{t('password')}</label>
              <input
                type="text"
                value={form.password}
                onChange={(event) => updateField('password', event.target.value)}
                placeholder={employee ? t('passwordKeepPlaceholder') : ''}
                required={!employee}
              />
            </div>
          </div>

          <label>{t('colOfficer')}</label>
          <input
            value={form.full_name}
            onChange={(event) => updateField('full_name', event.target.value)}
            required
          />

          <div className="form-row">
            <div>
              <label>{t('colRank')}</label>
              <input
                list="admin-ranks"
                value={form.rank}
                onChange={(event) => updateField('rank', event.target.value)}
              />
              <datalist id="admin-ranks">
                {RANKS.map((rank) => <option key={rank} value={rank} />)}
              </datalist>
            </div>
            <div>
              <label>{t('userRole')}</label>
              <select
                value={form.role}
                onChange={(event) => updateField('role', event.target.value)}
                required
              >
                <option value="officer">{t('role_officer')}</option>
                <option value="head">{t('role_head')}</option>
              </select>
            </div>
          </div>

          <div className="form-row">
            <div>
              <label>{t('colDistrict')}</label>
              <select
                value={form.district_id}
                onChange={(event) => updateField('district_id', event.target.value)}
                required={form.role === 'officer'}
              >
                <option value="">{form.role === 'head' ? t('allGroups') : t('selectGroup')}</option>
                {districts.map((district) => (
                  <option key={district.id} value={district.id}>
                    {districtName(district, lang)}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label>{t('sourceEmployeeNumber')}</label>
              <input
                value={form.source_employee_number}
                onChange={(event) => updateField('source_employee_number', event.target.value)}
              />
            </div>
            <div>
              <label>{t('monthlyNorm')}</label>
              <input
                type="number"
                min="1"
                step="1"
                value={form.monthly_norm}
                onChange={(event) => updateField('monthly_norm', event.target.value)}
                required
              />
            </div>
          </div>

          {error && <div className="error">{error}</div>}

          <div className="modal-actions">
            <button type="button" className="btn btn-ghost" onClick={onClose}>{t('cancel')}</button>
            <button type="submit" className="btn btn-primary" disabled={saving}>
              {t('save')}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

function CatalogModal({ item, onClose, onSaved }) {
  const { t } = useI18n()
  const { notify } = useNotify()
  const [form, setForm] = useState(() => catalogForm(item))
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const updateField = (field, value) => {
    setForm((current) => ({ ...current, [field]: value }))
  }

  const submit = async (event) => {
    event.preventDefault()
    setSaving(true)
    setError('')
    try {
      const payload = {
        code: form.code.trim(),
        indicator: form.indicator.trim(),
        block: form.block.trim(),
        subblock: form.subblock.trim(),
        npa: form.npa.trim(),
        points_per_unit: Number(form.points_per_unit),
        unit: form.unit.trim(),
        source_url: form.source_url.trim(),
        monthly_limit: Number(form.monthly_limit || 0),
        is_mass: Boolean(form.is_mass),
        is_critical: Boolean(form.is_critical),
      }
      const response = item
        ? await api.patch(`/admin/catalog/${item.id}`, payload)
        : await api.post('/admin/catalog', payload)
      onSaved(response.data)
      notify(t(item ? 'targetKpiSaved' : 'targetKpiCreated'))
      onClose()
    } catch (err) {
      const message = getApiErrorMessage(err, t, 'saveError')
      setError(message)
      notify(message, 'error')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal catalog-form-modal" onClick={(event) => event.stopPropagation()}>
        <h2>{t(item ? 'editTargetKpi' : 'addTargetKpi')}</h2>
        <form className="employee-form catalog-form" onSubmit={submit}>
          <div className="form-row">
            <div>
              <label>{t('colCode')}</label>
              <input
                value={form.code}
                onChange={(event) => updateField('code', event.target.value)}
                required
              />
            </div>
            <div>
              <label>{t('colBlock')}</label>
              <input
                value={form.block}
                onChange={(event) => updateField('block', event.target.value)}
                required
              />
            </div>
          </div>

          <label>{t('colIndicator')}</label>
          <textarea
            rows={3}
            value={form.indicator}
            onChange={(event) => updateField('indicator', event.target.value)}
            required
          />

          <div className="form-row">
            <div>
              <label>{t('colPointsPerUnit')}</label>
              <input
                type="number"
                min="0"
                step="0.01"
                value={form.points_per_unit}
                onChange={(event) => updateField('points_per_unit', event.target.value)}
                required
              />
            </div>
            <div>
              <label>{t('monthlyLimit')}</label>
              <input
                type="number"
                min="0"
                step="0.01"
                value={form.monthly_limit}
                onChange={(event) => updateField('monthly_limit', event.target.value)}
              />
            </div>
            <div>
              <label>{t('colUnit')}</label>
              <input
                value={form.unit}
                onChange={(event) => updateField('unit', event.target.value)}
              />
            </div>
          </div>

          <div className="form-row">
            <div>
              <label>{t('colSubblock')}</label>
              <input
                value={form.subblock}
                onChange={(event) => updateField('subblock', event.target.value)}
              />
            </div>
            <div>
              <label>{t('colNpa')}</label>
              <input
                value={form.npa}
                onChange={(event) => updateField('npa', event.target.value)}
              />
            </div>
          </div>

          <label>{t('colSource')}</label>
          <input
            value={form.source_url}
            onChange={(event) => updateField('source_url', event.target.value)}
          />

          <div className="catalog-check-row">
            <label>
              <input
                type="checkbox"
                checked={form.is_mass}
                onChange={(event) => updateField('is_mass', event.target.checked)}
              />
              {t('massIndicator')}
            </label>
            <label>
              <input
                type="checkbox"
                checked={form.is_critical}
                onChange={(event) => updateField('is_critical', event.target.checked)}
              />
              {t('criticalIndicator')}
            </label>
          </div>

          {error && <div className="error">{error}</div>}

          <div className="modal-actions">
            <button type="button" className="btn btn-ghost" onClick={onClose}>{t('cancel')}</button>
            <button type="submit" className="btn btn-primary" disabled={saving}>
              {t('save')}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default function Admin({ initialTab = 'employees' }) {
  const { t, fmtMonth, lang } = useI18n()
  const { notify } = useNotify()
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState(initialTab)
  const [employees, setEmployees] = useState([])
  const [catalogItems, setCatalogItems] = useState([])
  const [ratingRows, setRatingRows] = useState([])
  const [months, setMonths] = useState([])
  const [districts, setDistricts] = useState([])
  const [query, setQuery] = useState('')
  const [catalogQuery, setCatalogQuery] = useState('')
  const [districtFilter, setDistrictFilter] = useState('')
  const [blockFilter, setBlockFilter] = useState('')
  const [ratingMonth, setRatingMonth] = useState('')
  const [ratingDistrictFilter, setRatingDistrictFilter] = useState('')
  const [ratingOfficerFilter, setRatingOfficerFilter] = useState('')
  const [ratingCategoryFilter, setRatingCategoryFilter] = useState('')
  const [ratingSort, setRatingSort] = useState({ key: 'score', dir: 'desc' })
  const [loading, setLoading] = useState(true)
  const [ratingLoading, setRatingLoading] = useState(false)
  const [editingEmployee, setEditingEmployee] = useState(null)
  const [editingCatalogItem, setEditingCatalogItem] = useState(null)
  const [showCreate, setShowCreate] = useState(false)
  const [showCatalogCreate, setShowCatalogCreate] = useState(false)

  const loadAdminData = useCallback(async () => {
    setLoading(true)
    try {
      const [usersResponse, districtsResponse, catalogResponse, monthsResponse] = await Promise.all([
        api.get('/admin/users'),
        api.get('/districts'),
        api.get('/admin/catalog'),
        api.get('/months'),
      ])
      setEmployees(Array.isArray(usersResponse.data) ? usersResponse.data : [])
      setDistricts(Array.isArray(districtsResponse.data) ? districtsResponse.data : [])
      setCatalogItems(Array.isArray(catalogResponse.data) ? catalogResponse.data : [])
      setMonths(Array.isArray(monthsResponse.data) ? monthsResponse.data : [])
    } catch (err) {
      notify(getApiErrorMessage(err, t, 'adminDataLoadError'), 'error')
    } finally {
      setLoading(false)
    }
  }, [notify, t])

  const loadRatingData = useCallback(async () => {
    setRatingLoading(true)
    try {
      const params = {}
      if (ratingMonth) params.month = ratingMonth
      if (ratingDistrictFilter) params.district_id = ratingDistrictFilter
      const { data } = await api.get('/admin/rating', { params })
      const rows = (Array.isArray(data) ? data : [])
        .map((officer) => {
          const finalScore = Number(officer.final) || 0
          return {
            officer_id: officer.officer_id,
            name: officer.name,
            rank: officer.rank,
            district: officer.district || '',
            district_id: officer.district_id || '',
            final: finalScore,
            confirmed: Number(officer.confirmed) || 0,
            pending: Number(officer.pending) || 0,
            category_score: finalScore,
            efficiency_rank: officer.efficiency_rank,
            efficiency_total: officer.efficiency_total,
            efficiency_score: officer.efficiency_score,
          }
        })
        .sort((left, right) => (right.final - left.final) || compareRatingNames(left, right))
      setRatingRows(rows)
    } catch (err) {
      notify(getApiErrorMessage(err, t, 'ratingLoadError'), 'error')
      setRatingRows([])
    } finally {
      setRatingLoading(false)
    }
  }, [notify, ratingDistrictFilter, ratingMonth, t])

  useEffect(() => {
    loadAdminData()
  }, [loadAdminData])

  useEffect(() => {
    if (activeTab === 'rating') loadRatingData()
  }, [activeTab, loadRatingData])

  useEffect(() => {
    setActiveTab(initialTab)
  }, [initialTab])

  const filteredEmployees = useMemo(() => {
    const needle = query.trim().toLocaleLowerCase()
    return employees
      .filter((employee) => (
        !districtFilter || String(employee.district?.id || '') === String(districtFilter)
      ))
      .filter((employee) => {
        if (!needle) return true
        return [
          employee.full_name,
          employee.username,
          employee.rank,
          employee.source_employee_number,
          t(`role_${employee.role}`),
          districtName(employee.district, lang),
        ].some((value) => (value || '').toLocaleLowerCase().includes(needle))
      })
  }, [districtFilter, employees, lang, query])

  const catalogBlocks = useMemo(() => (
    Array.from(new Set(catalogItems.map((item) => item.block).filter(Boolean)))
      .sort((left, right) => left.localeCompare(right, 'ru'))
  ), [catalogItems])

  const filteredCatalogItems = useMemo(() => {
    const needle = catalogQuery.trim().toLocaleLowerCase()
    return catalogItems
      .filter((item) => !blockFilter || item.block === blockFilter)
      .filter((item) => {
        if (!needle) return true
        return [
          item.code,
          item.indicator,
          item.block,
          item.subblock,
          item.npa,
          item.unit,
        ].some((value) => (value || '').toLocaleLowerCase().includes(needle))
      })
  }, [blockFilter, catalogItems, catalogQuery])

  const displayedRatingMonths = ratingMonth && !months.includes(ratingMonth)
    ? [ratingMonth, ...months]
    : months

  const ratingOfficerOptions = useMemo(() => (
    ratingRows.map((officer) => ({
      id: officer.officer_id,
      label: officer.name,
      description: [officer.rank, districtName(officer.district, lang)].filter(Boolean).join(' · '),
    }))
  ), [lang, ratingRows])

  const categorizedRatingRows = useMemo(() => {
    const maxScore = maxEffectivenessScore(ratingRows)
    return ratingRows.map((officer) => ({
      ...officer,
      achievement_percent: relativeScorePercent(officer.final, maxScore),
      effectivenessCategory: effectivenessCategory(officer, maxScore),
    }))
  }, [ratingRows])

  const filteredRatingRows = useMemo(() => {
    const rows = categorizedRatingRows
      .map((officer, index) => ({ ...officer, rankNum: index + 1 }))
      .filter((officer) => (
        !ratingOfficerFilter || String(officer.officer_id) === String(ratingOfficerFilter)
      ))
      .filter((officer) => (
        !ratingCategoryFilter || officer.effectivenessCategory === ratingCategoryFilter
      ))

    const factor = ratingSort.dir === 'asc' ? 1 : -1
    return [...rows].sort((left, right) => {
      const comparison = ratingSort.key === 'score'
        ? left.final - right.final
        : ratingSort.key === 'rank'
          ? rankOrder(left.rank) - rankOrder(right.rank)
          : compareRatingNames(left, right)
      return comparison ? comparison * factor : compareRatingNames(left, right)
    })
  }, [categorizedRatingRows, ratingCategoryFilter, ratingOfficerFilter, ratingSort])

  const upsertEmployee = (employee) => {
    setEmployees((current) => {
      if (current.some((item) => item.id === employee.id)) {
        return current
          .map((item) => (item.id === employee.id ? employee : item))
          .sort((left, right) => left.full_name.localeCompare(right.full_name, 'ru'))
      }
      return [...current, employee].sort((left, right) => left.full_name.localeCompare(right.full_name, 'ru'))
    })
  }

  const upsertCatalogItem = (catalogItem) => {
    setCatalogItems((current) => {
      if (current.some((item) => item.id === catalogItem.id)) {
        return current
          .map((item) => (item.id === catalogItem.id ? catalogItem : item))
          .sort((left, right) => left.code.localeCompare(right.code, 'ru'))
      }
      return [...current, catalogItem].sort((left, right) => left.code.localeCompare(right.code, 'ru'))
    })
  }

  const deleteEmployee = async (employee) => {
    const message = employee.entries_count > 0
      ? formatTemplate(t('confirmDeleteEmployeeWithEntries'), {
        name: employee.full_name,
        count: employee.entries_count,
      })
      : formatTemplate(t('confirmDeleteEmployee'), { name: employee.full_name })
    if (!window.confirm(message)) return
    try {
      await api.delete(`/admin/users/${employee.id}`)
      setEmployees((current) => current.filter((item) => item.id !== employee.id))
      notify(t('employeeDeleted'))
    } catch (err) {
      notify(getApiErrorMessage(err, t, 'employeeDeleteError'), 'error')
    }
  }

  const selectAdminTab = (tab) => {
    setActiveTab(tab)
    navigate({
      employees: '/admin',
      catalog: '/admin/catalog',
      rating: '/admin/rating',
      support: '/admin/support',
    }[tab])
  }

  const toggleRatingSort = (key) => {
    setRatingSort((current) => (
      current.key === key
        ? { key, dir: current.dir === 'asc' ? 'desc' : 'asc' }
        : { key, dir: 'asc' }
    ))
  }

  const deleteCatalogItem = async (item) => {
    const message = formatTemplate(t('confirmDeleteTargetKpi'), { code: item.code })
    if (!window.confirm(message)) return
    try {
      await api.delete(`/admin/catalog/${item.id}`)
      setCatalogItems((current) => current.filter((catalogItem) => catalogItem.id !== item.id))
      notify(t('targetKpiDeleted'))
    } catch (err) {
      notify(getApiErrorMessage(err, t, 'catalogDeleteError'), 'error')
    }
  }

  return (
    <div>
      <div className="page-head admin-page-head">
        <h1>{t('adminPanel')}</h1>
        <div className="admin-tabs" role="tablist" aria-label={t('adminPanel')}>
          <button
            type="button"
            className={`btn ${activeTab === 'employees' ? 'btn-primary' : 'btn-ghost'}`}
            onClick={() => selectAdminTab('employees')}
          >
            {t('employees')}
          </button>
          <button
            type="button"
            className={`btn ${activeTab === 'catalog' ? 'btn-primary' : 'btn-ghost'}`}
            onClick={() => selectAdminTab('catalog')}
          >
            {t('targets')}
          </button>
          <button
            type="button"
            className={`btn ${activeTab === 'rating' ? 'btn-primary' : 'btn-ghost'}`}
            onClick={() => selectAdminTab('rating')}
          >
            {t('officers')}
          </button>
          <button
            type="button"
            className={`btn ${activeTab === 'support' ? 'btn-primary' : 'btn-ghost'}`}
            onClick={() => selectAdminTab('support')}
          >
            {t('supportChat')}
          </button>
        </div>
      </div>

      {activeTab === 'employees' && (
        <>
          <div className="admin-section-toolbar">
            <input
              className="admin-search-input"
              type="search"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder={t('searchEmployee')}
              aria-label={t('searchEmployee')}
            />
            <select
              className="select admin-district-select"
              value={districtFilter}
              onChange={(event) => setDistrictFilter(event.target.value)}
              aria-label={t('selectGroup')}
            >
              <option value="">{t('allGroups')}</option>
              {districts.map((district) => (
                <option key={district.id} value={district.id}>
                  {districtName(district, lang)}
                </option>
              ))}
            </select>
            <button
              type="button"
              className="btn btn-primary"
              onClick={() => setShowCreate(true)}
              disabled={!districts.length}
            >
              + {t('addEmployee')}
            </button>
          </div>

          <div className="card table-card">
            <div className="table-scroll admin-table-scroll">
              <table className="data-table admin-table">
                <colgroup>
                  <col className="admin-col-name" />
                  <col className="admin-col-login" />
                  <col className="admin-col-role" />
                  <col className="admin-col-rank" />
                  <col className="admin-col-district" />
                  <col className="admin-col-norm" />
                  <col className="admin-col-number" />
                  <col className="admin-col-records" />
                  <col className="admin-col-actions" />
                </colgroup>
                <thead>
                  <tr>
                    <th>{t('colOfficer')}</th>
                    <th>{t('username')}</th>
                    <th>{t('userRole')}</th>
                    <th>{t('colRank')}</th>
                    <th>{t('colDistrict')}</th>
                    <th className="num">{t('monthlyNorm')}</th>
                    <th>{t('sourceEmployeeNumber')}</th>
                    <th className="num">{t('entriesCount')}</th>
                    <th>{t('colActions')}</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredEmployees.map((employee) => (
                    <tr key={employee.id}>
                      <td className="strong">{employee.full_name}</td>
                      <td>{employee.username}</td>
                      <td>{t(`role_${employee.role}`)}</td>
                      <td className="muted">{employee.rank || '—'}</td>
                      <td className="muted">{districtName(employee.district, lang) || '—'}</td>
                      <td className="num">{employee.monthly_norm}</td>
                      <td className="muted">{employee.source_employee_number || '—'}</td>
                      <td className="num">{employee.entries_count}</td>
                      <td>
                        <div className="admin-action-group">
                          <button
                            type="button"
                            className="btn btn-sm btn-ghost"
                            onClick={() => setEditingEmployee(employee)}
                          >
                            {t('edit')}
                          </button>
                          <button
                            type="button"
                            className="btn btn-sm btn-delete"
                            onClick={() => deleteEmployee(employee)}
                          >
                            {t('delete')}
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                  {!loading && filteredEmployees.length === 0 && (
                    <tr>
                      <td colSpan={9} className="empty">
                        {employees.length === 0 ? t('noEmployees') : t('nothingFound')}
                      </td>
                    </tr>
                  )}
                  {loading && (
                    <tr>
                      <td colSpan={9} className="empty">{t('loadingEntries')}</td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </>
      )}

      {activeTab === 'catalog' && (
        <>
          <div className="admin-section-toolbar">
            <input
              className="admin-search-input"
              type="search"
              value={catalogQuery}
              onChange={(event) => setCatalogQuery(event.target.value)}
              placeholder={t('searchTargetKpi')}
              aria-label={t('searchTargetKpi')}
            />
            <select
              className="select admin-block-select"
              value={blockFilter}
              onChange={(event) => setBlockFilter(event.target.value)}
              aria-label={t('allBlocks')}
            >
              <option value="">{t('allBlocks')}</option>
              {catalogBlocks.map((block) => (
                <option key={block} value={block}>{block}</option>
              ))}
            </select>
            <button
              type="button"
              className="btn btn-primary"
              onClick={() => setShowCatalogCreate(true)}
            >
              + {t('addTargetKpi')}
            </button>
          </div>

          <div className="card table-card">
            <div className="table-scroll admin-table-scroll">
              <table className="data-table admin-catalog-table">
                <colgroup>
                  <col className="admin-col-kpi-code" />
                  <col className="admin-col-kpi-indicator" />
                  <col className="admin-col-kpi-block" />
                  <col className="admin-col-kpi-unit" />
                  <col className="admin-col-kpi-points" />
                  <col className="admin-col-kpi-limit" />
                  <col className="admin-col-kpi-flags" />
                  <col className="admin-col-kpi-actions" />
                </colgroup>
                <thead>
                  <tr>
                    <th>{t('colCode')}</th>
                    <th>{t('colIndicator')}</th>
                    <th>{t('colBlock')}</th>
                    <th>{t('colUnit')}</th>
                    <th className="num">{t('colPointsPerUnit')}</th>
                    <th className="num">{t('monthlyLimit')}</th>
                    <th>{t('flags')}</th>
                    <th>{t('colActions')}</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredCatalogItems.map((item) => (
                    <tr key={item.id}>
                      <td><span className="code-chip">{item.code}</span></td>
                      <td>
                        <div className="admin-catalog-indicator">{item.indicator}</div>
                        {item.npa && <div className="muted">{item.npa}</div>}
                      </td>
                      <td className="muted">{item.block}</td>
                      <td className="muted">{item.unit}</td>
                      <td className="num">{item.points_per_unit}</td>
                      <td className="num">{item.monthly_limit || '—'}</td>
                      <td className="muted">
                        {[item.is_mass ? t('massIndicator') : '', item.is_critical ? t('criticalIndicator') : '']
                          .filter(Boolean)
                          .join(', ') || '—'}
                      </td>
                      <td>
                        <div className="admin-action-group">
                          <button
                            type="button"
                            className="btn btn-sm btn-ghost"
                            onClick={() => setEditingCatalogItem(item)}
                          >
                            {t('edit')}
                          </button>
                          <button
                            type="button"
                            className="btn btn-sm btn-delete"
                            onClick={() => deleteCatalogItem(item)}
                          >
                            {t('delete')}
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                  {!loading && filteredCatalogItems.length === 0 && (
                    <tr>
                      <td colSpan={8} className="empty">
                        {catalogItems.length === 0 ? t('noTargetKpis') : t('nothingFound')}
                      </td>
                    </tr>
                  )}
                  {loading && (
                    <tr>
                      <td colSpan={8} className="empty">{t('loadingTargets')}</td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </>
      )}

      {activeTab === 'rating' && (
        <>
          <div className="admin-section-toolbar admin-rating-toolbar">
            <NameFilterCombobox
              items={ratingOfficerOptions}
              value={ratingOfficerFilter}
              onChange={setRatingOfficerFilter}
            />
            <select
              className="select officers-period-select"
              value={ratingMonth}
              onChange={(event) => setRatingMonth(event.target.value)}
              aria-label={t('period')}
            >
              <option value="">{t('periodAll')}</option>
              {displayedRatingMonths.map((item) => (
                <option key={item} value={item}>{fmtMonth(item)}</option>
              ))}
            </select>
            <select
              className="select officers-district-select"
              value={ratingDistrictFilter}
              onChange={(event) => {
                setRatingDistrictFilter(event.target.value)
                setRatingOfficerFilter('')
              }}
              aria-label={t('selectGroup')}
            >
              <option value="">{t('allGroups')}</option>
              {districts.map((district) => (
                <option key={district.id} value={district.id}>
                  {districtName(district, lang)}
                </option>
              ))}
            </select>
            <select
              className="select officers-category-select"
              value={ratingCategoryFilter}
              onChange={(event) => setRatingCategoryFilter(event.target.value)}
              aria-label={t('effectivenessCategory')}
            >
              <option value="">{t('allEffectivenessCategories')}</option>
              {EFFECTIVENESS_CATEGORIES.map((item) => (
                <option key={item} value={item}>
                  {t({
                    very_effective: 'categoryVeryEffective',
                    effective: 'categoryEffective',
                    little_effective: 'categoryLittleEffective',
                  }[item])}
                </option>
              ))}
            </select>
            <select
              className="select officers-score-sort"
              value={ratingSort.key === 'score' ? ratingSort.dir : ''}
              onChange={(event) => setRatingSort({ key: 'score', dir: event.target.value })}
              aria-label={t('sortEffectiveness')}
            >
              {ratingSort.key !== 'score' && <option value="">{t('sortEffectiveness')}</option>}
              <option value="desc">{t('sortBestFirst')}</option>
              <option value="asc">{t('sortWorstFirst')}</option>
            </select>
          </div>

          <div className="card table-card">
            <div className="table-scroll admin-table-scroll">
              <table className="data-table admin-rating-table">
                <colgroup>
                  <col className="admin-rating-col-position" />
                  <col className="admin-rating-col-name" />
                  <col className="admin-rating-col-rank" />
                  <col className="admin-rating-col-district" />
                  <col className="admin-rating-col-score" />
                  <col className="admin-rating-col-achievement" />
                  <col className="admin-rating-col-category" />
                  <col className="admin-rating-col-confirmed" />
                  <col className="admin-rating-col-pending" />
                </colgroup>
                <thead>
                  <tr>
                    <th>#</th>
                    <SortTh column="surname" label={t('colOfficer')} sort={ratingSort} onSort={toggleRatingSort} />
                    <SortTh column="rank" label={t('colRank')} sort={ratingSort} onSort={toggleRatingSort} />
                    <th>{t('colDistrict')}</th>
                    <th className="num">{t('confirmedFinal')}</th>
                    <th className="num">{t('colAchievement')}</th>
                    <th>{t('effectivenessCategory')}</th>
                    <th className="num">{t('confirmedCount')}</th>
                    <th className="num">{t('pendingShort')}</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredRatingRows.map((officer) => (
                    <tr key={officer.officer_id}>
                      <td className="rank-cell">
                        <span className={`rank-num rank-${officer.rankNum}${officer.rankNum >= 100 ? ' rank-num-wide' : ''}`}>
                          {officer.rankNum}
                        </span>
                      </td>
                      <td className="strong">{officer.name}</td>
                      <td className="muted">{officer.rank || '—'}</td>
                      <td className="muted">{districtName(officer.district, lang) || '—'}</td>
                      <td className="num strong">{officer.final}</td>
                      <td className="num">{officer.achievement_percent}%</td>
                      <td>
                        <EffectivenessBadge category={officer.effectivenessCategory} t={t} />
                      </td>
                      <td className="num">{officer.confirmed}</td>
                      <td className="num">{officer.pending}</td>
                    </tr>
                  ))}
                  {!ratingLoading && filteredRatingRows.length === 0 && (
                    <tr>
                      <td colSpan={9} className="empty">
                        {ratingRows.length === 0 ? t('noEmployees') : t('nothingFound')}
                      </td>
                    </tr>
                  )}
                  {ratingLoading && (
                    <tr>
                      <td colSpan={9} className="empty">{t('loadingEntries')}</td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </>
      )}

      {activeTab === 'support' && <AdminSupportChat />}

      {showCreate && (
        <EmployeeModal
          districts={districts}
          onClose={() => setShowCreate(false)}
          onSaved={upsertEmployee}
        />
      )}
      {editingEmployee && (
        <EmployeeModal
          employee={editingEmployee}
          districts={districts}
          onClose={() => setEditingEmployee(null)}
          onSaved={upsertEmployee}
        />
      )}
      {showCatalogCreate && (
        <CatalogModal
          onClose={() => setShowCatalogCreate(false)}
          onSaved={upsertCatalogItem}
        />
      )}
      {editingCatalogItem && (
        <CatalogModal
          item={editingCatalogItem}
          onClose={() => setEditingCatalogItem(null)}
          onSaved={upsertCatalogItem}
        />
      )}
    </div>
  )
}
