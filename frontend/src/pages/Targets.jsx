import { useEffect, useMemo, useState } from 'react'
import api from '../api'
import { getApiErrorMessage } from '../apiError.js'
import { useI18n } from '../i18n.jsx'
import { useNotify } from '../notifications.jsx'

const currentMonth = () => new Date().toISOString().slice(0, 7)

const toEditableRows = (rows) => rows.map((row) => ({
  ...row,
  target_quantity: row.mixed ? '' : (row.target_quantity ? String(row.target_quantity) : ''),
  dirty: false,
}))

export default function Targets() {
  const { t, fmtMonth } = useI18n()
  const { notify } = useNotify()
  const [officers, setOfficers] = useState([])
  const [districts, setDistricts] = useState([])
  const [months, setMonths] = useState([])
  const [scope, setScope] = useState('officer')
  const [officerId, setOfficerId] = useState('')
  const [districtId, setDistrictId] = useState('')
  const [month, setMonth] = useState(currentMonth())
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  const selectedOfficer = useMemo(
    () => officers.find((item) => String(item.id) === String(officerId)),
    [officers, officerId]
  )
  const selectedDistrict = useMemo(
    () => districts.find((item) => String(item.id) === String(districtId)),
    [districts, districtId]
  )
  const coeff = scope === 'group'
    ? selectedDistrict?.coeff ?? 1
    : selectedOfficer?.district?.coeff ?? 1
  const selectionId = scope === 'group' ? districtId : officerId

  const loadTargets = async (selectedMonth, selectedScope, selectedOfficerId, selectedDistrictId) => {
    const selectedId = selectedScope === 'group' ? selectedDistrictId : selectedOfficerId
    if (!selectedId) {
      setRows([])
      setLoading(false)
      return
    }
    setLoading(true)
    try {
      const endpoint = selectedScope === 'group' ? '/group-targets' : '/targets'
      const params = selectedScope === 'group'
        ? { month: selectedMonth, district_id: Number(selectedDistrictId) }
        : { month: selectedMonth, officer_id: Number(selectedOfficerId) }
      const { data } = await api.get(endpoint, {
        params,
      })
      setRows(toEditableRows(data))
    } catch (err) {
      notify(getApiErrorMessage(err, t, 'saveError'), 'error')
      setRows([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    api.get('/officers')
      .then(({ data }) => {
        setOfficers(data)
        setOfficerId((prev) => prev || String(data[0]?.id || ''))
      })
      .catch((err) => notify(getApiErrorMessage(err, t, 'saveError'), 'error'))

    api.get('/districts')
      .then(({ data }) => {
        setDistricts(data)
        setDistrictId((prev) => prev || String(data[0]?.id || ''))
      })
      .catch((err) => notify(getApiErrorMessage(err, t, 'saveError'), 'error'))

    api.get('/months')
      .then(({ data }) => {
        const list = [...new Set([currentMonth(), ...data])].sort().reverse()
        setMonths(list)
        setMonth((prev) => (list.includes(prev) ? prev : list[0] || currentMonth()))
      })
      .catch((err) => notify(getApiErrorMessage(err, t, 'saveError'), 'error'))
  }, [])

  useEffect(() => {
    loadTargets(month, scope, officerId, districtId)
  }, [month, scope, officerId, districtId])

  const saveTargets = async () => {
    if (!selectionId) return
    setSaving(true)
    try {
      const items = rows
        .filter((row) => scope !== 'group' || !row.mixed || row.dirty)
        .map((row) => ({
          catalog_id: row.catalog_id,
          target_quantity: Number(row.target_quantity || 0),
        }))
      const payload = {
        ...(scope === 'group'
          ? { district_id: Number(districtId) }
          : { officer_id: Number(officerId) }),
        month,
        items,
      }
      const endpoint = scope === 'group' ? '/group-targets' : '/targets'
      const { data } = await api.put(endpoint, payload)
      setRows(toEditableRows(data))
      notify(t(scope === 'group' ? 'groupTargetsSaved' : 'targetsSaved'))
    } catch (err) {
      notify(getApiErrorMessage(err, t, 'saveError'), 'error')
    } finally {
      setSaving(false)
    }
  }

  const updateQuantity = (catalogId, value) => {
    setRows((prev) => prev.map((row) => (
      row.catalog_id === catalogId
        ? { ...row, target_quantity: value, mixed: false, dirty: true }
        : row
    )))
  }

  return (
    <div>
      <div className="page-head">
        <h1>{t('targets')}</h1>
        <div className="page-head-actions">
          <select
            className="select"
            aria-label={t('targetMonth')}
            title={t('targetMonth')}
            value={month}
            onChange={(e) => setMonth(e.target.value)}
          >
            {months.map((item) => (
              <option key={item} value={item}>{fmtMonth(item)}</option>
            ))}
          </select>
          <select
            className="select"
            aria-label={t('targetScope')}
            title={t('targetScope')}
            value={scope}
            onChange={(e) => setScope(e.target.value)}
          >
            <option value="officer">{t('targetScopeOfficer')}</option>
            <option value="group">{t('targetScopeGroup')}</option>
          </select>
          {scope === 'officer' ? (
            <select
              className="select employee-select"
              aria-label={t('selectOfficer')}
              title={t('selectOfficer')}
              value={officerId}
              onChange={(e) => setOfficerId(e.target.value)}
            >
              {officers.map((officer) => (
                <option key={officer.id} value={officer.id}>
                  {officer.full_name}
                </option>
              ))}
            </select>
          ) : (
            <select
              className="select employee-select"
              aria-label={t('selectGroup')}
              title={t('selectGroup')}
              value={districtId}
              onChange={(e) => setDistrictId(e.target.value)}
            >
              {districts.map((district) => (
                <option key={district.id} value={district.id}>
                  {district.name}
                </option>
              ))}
            </select>
          )}
          <button className="btn btn-primary" onClick={saveTargets} disabled={saving || !selectionId}>
            {t('save')}
          </button>
        </div>
      </div>

      <div className="card table-card">
        <div className="table-scroll">
          <table className="data-table">
            <thead>
              <tr>
                <th>{t('colCode')}</th>
                <th>{t('colIndicator')}</th>
                <th>{t('colUnit')}</th>
                <th className="num">{t('colPointsPerUnit')}</th>
                <th className="num">{t('colTargetQuantity')}</th>
                <th className="num">{t('colTargetScore')}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => {
                const targetQuantity = Number(row.target_quantity || 0)
                const targetScore = Math.round(targetQuantity * row.points_per_unit * coeff * 100) / 100
                return (
                  <tr key={row.catalog_id}>
                    <td><span className="code-chip">{row.code}</span></td>
                    <td>{row.indicator}</td>
                    <td className="muted">{row.unit}</td>
                    <td className="num">{row.points_per_unit}</td>
                    <td className="num target-cell">
                      <input
                        className="table-number-input"
                        type="number"
                        min="0"
                        step="0.5"
                        value={row.target_quantity}
                        placeholder={row.mixed ? t('mixedTargets') : ''}
                        title={row.mixed ? t('mixedTargets') : undefined}
                        onChange={(e) => updateQuantity(row.catalog_id, e.target.value)}
                      />
                    </td>
                    <td className="num strong">{row.mixed ? '—' : targetScore}</td>
                  </tr>
                )
              })}
              {!loading && rows.length === 0 && (
                <tr><td colSpan={6} className="empty">{t('nothingFound')}</td></tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
