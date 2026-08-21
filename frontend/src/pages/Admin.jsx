import { useCallback, useEffect, useMemo, useState } from 'react'
import api from '../api'
import { getApiErrorMessage } from '../apiError.js'
import { districtName } from '../districtLocale.js'
import { useI18n } from '../i18n.jsx'
import { useNotify } from '../notifications.jsx'

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

export default function Admin() {
  const { t, lang } = useI18n()
  const { notify } = useNotify()
  const [employees, setEmployees] = useState([])
  const [districts, setDistricts] = useState([])
  const [query, setQuery] = useState('')
  const [districtFilter, setDistrictFilter] = useState('')
  const [loading, setLoading] = useState(true)
  const [editingEmployee, setEditingEmployee] = useState(null)
  const [showCreate, setShowCreate] = useState(false)

  const loadAdminData = useCallback(async () => {
    setLoading(true)
    try {
      const [usersResponse, districtsResponse] = await Promise.all([
        api.get('/admin/users'),
        api.get('/districts'),
      ])
      setEmployees(Array.isArray(usersResponse.data) ? usersResponse.data : [])
      setDistricts(Array.isArray(districtsResponse.data) ? districtsResponse.data : [])
    } catch (err) {
      notify(getApiErrorMessage(err, t, 'adminUsersLoadError'), 'error')
    } finally {
      setLoading(false)
    }
  }, [notify, t])

  useEffect(() => {
    loadAdminData()
  }, [loadAdminData])

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

  return (
    <div>
      <div className="page-head admin-page-head">
        <h1>{t('adminPanel')}</h1>
        <div className="page-head-actions admin-toolbar">
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
    </div>
  )
}
