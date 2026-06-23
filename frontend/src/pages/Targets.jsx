import { useEffect, useMemo, useState } from 'react'
import api from '../api'
import { getApiErrorMessage } from '../apiError.js'
import { useI18n } from '../i18n.jsx'
import { useNotify } from '../notifications.jsx'

const currentMonth = () => new Date().toISOString().slice(0, 7)

const toEditableRows = (rows) => rows.map((row) => ({
  ...row,
  target_quantity: row.target_quantity ? String(row.target_quantity) : '',
}))

export default function Targets() {
  const { t, fmtMonth } = useI18n()
  const { notify } = useNotify()
  const [officers, setOfficers] = useState([])
  const [months, setMonths] = useState([])
  const [officerId, setOfficerId] = useState('')
  const [month, setMonth] = useState(currentMonth())
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  const selectedOfficer = useMemo(
    () => officers.find((item) => String(item.id) === String(officerId)),
    [officers, officerId]
  )
  const coeff = selectedOfficer?.district?.coeff ?? 1

  const loadTargets = async (selectedMonth, selectedOfficerId) => {
    if (!selectedOfficerId) {
      setRows([])
      setLoading(false)
      return
    }
    setLoading(true)
    try {
      const { data } = await api.get('/targets', {
        params: { month: selectedMonth, officer_id: Number(selectedOfficerId) },
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

    api.get('/months')
      .then(({ data }) => {
        const list = [...new Set([currentMonth(), ...data])].sort().reverse()
        setMonths(list)
        setMonth((prev) => (list.includes(prev) ? prev : list[0] || currentMonth()))
      })
      .catch((err) => notify(getApiErrorMessage(err, t, 'saveError'), 'error'))
  }, [])

  useEffect(() => {
    loadTargets(month, officerId)
  }, [month, officerId])

  const saveTargets = async () => {
    if (!officerId) return
    setSaving(true)
    try {
      const payload = {
        officer_id: Number(officerId),
        month,
        items: rows.map((row) => ({
          catalog_id: row.catalog_id,
          target_quantity: Number(row.target_quantity || 0),
        })),
      }
      const { data } = await api.put('/targets', payload)
      setRows(toEditableRows(data))
      notify(t('targetsSaved'))
    } catch (err) {
      notify(getApiErrorMessage(err, t, 'saveError'), 'error')
    } finally {
      setSaving(false)
    }
  }

  const updateQuantity = (catalogId, value) => {
    setRows((prev) => prev.map((row) => (
      row.catalog_id === catalogId ? { ...row, target_quantity: value } : row
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
          <button className="btn btn-primary" onClick={saveTargets} disabled={saving || !officerId}>
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
                        onChange={(e) => updateQuantity(row.catalog_id, e.target.value)}
                      />
                    </td>
                    <td className="num strong">{targetScore}</td>
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
