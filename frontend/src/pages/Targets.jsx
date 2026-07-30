import { useEffect, useMemo, useState } from 'react'
import api from '../api'
import { getApiErrorMessage } from '../apiError.js'
import KpiCodeCombobox from '../components/KpiCodeCombobox.jsx'
import NameFilterCombobox from '../components/NameFilterCombobox.jsx'
import { useAuth } from '../auth.jsx'
import { districtName } from '../districtLocale.js'
import { useI18n } from '../i18n.jsx'
import { kpiIndicator } from '../kpiLocale.js'
import { useNotify } from '../notifications.jsx'

const currentMonth = () => new Date().toISOString().slice(0, 7)

const toEditableRows = (rows) => rows.map((row) => ({
  ...row,
  target_quantity: row.mixed ? '' : (row.target_quantity ? String(row.target_quantity) : ''),
  dirty: false,
}))

export default function Targets() {
  const { t, fmtMonth, lang } = useI18n()
  const { user, isHead } = useAuth()
  const { notify } = useNotify()
  const [officers, setOfficers] = useState([])
  const [districts, setDistricts] = useState([])
  const [months, setMonths] = useState([])
  const [scope, setScope] = useState('officer')
  const [officerId, setOfficerId] = useState('')
  const [districtId, setDistrictId] = useState('')
  const [month, setMonth] = useState(currentMonth())
  const [rows, setRows] = useState([])
  const [kpiFilter, setKpiFilter] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  const selectedOfficer = useMemo(
    () => officers.find((item) => String(item.id) === String(officerId)),
    [officers, officerId]
  )
  const officerOptions = useMemo(
    () => officers.map((officer) => ({
      id: officer.id,
      label: officer.full_name,
      description: [
        officer.rank,
        districtName(officer.district || officer.district_name, lang),
      ].filter(Boolean).join(' · '),
    })),
    [lang, officers]
  )
  const selectedDistrict = useMemo(
    () => districts.find((item) => String(item.id) === String(districtId)),
    [districts, districtId]
  )
  const coeff = isHead
    ? scope === 'group'
      ? selectedDistrict?.coeff ?? 1
      : selectedOfficer?.district?.coeff ?? 1
    : user?.district?.coeff ?? 1
  const selectionId = isHead
    ? scope === 'group' ? districtId : officerId
    : user?.id
  const targetRows = useMemo(
    () => isHead
      ? rows
      : rows.filter((row) => Number(row.target_quantity) > 0 || Number(row.target_score) > 0),
    [isHead, rows]
  )
  const kpiOptions = useMemo(
    () => targetRows.map((row) => ({
      id: row.catalog_id,
      code: row.code,
      indicator: row.indicator,
      points_per_unit: row.points_per_unit,
      unit: row.unit,
    })),
    [targetRows]
  )
  const visibleRows = useMemo(
    () => targetRows.filter((row) => !kpiFilter || String(row.catalog_id) === String(kpiFilter)),
    [kpiFilter, targetRows]
  )

  const loadTargets = async (selectedMonth, selectedScope, selectedOfficerId, selectedDistrictId) => {
    const selectedId = selectedScope === 'group' ? selectedDistrictId : selectedOfficerId
    if (isHead && !selectedId) {
      setRows([])
      setLoading(false)
      return
    }
    setLoading(true)
    try {
      const endpoint = isHead && selectedScope === 'group' ? '/group-targets' : '/targets'
      const params = isHead
        ? selectedScope === 'group'
          ? { month: selectedMonth, district_id: Number(selectedDistrictId) }
          : { month: selectedMonth, officer_id: Number(selectedOfficerId) }
        : { month: selectedMonth }
      const { data } = await api.get(endpoint, {
        params,
      })
      setRows(toEditableRows(data))
      setKpiFilter((current) => (
        current && !data.some((row) => String(row.catalog_id) === String(current))
          ? ''
          : current
      ))
    } catch (err) {
      notify(getApiErrorMessage(err, t, 'saveError'), 'error')
      setRows([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (isHead) {
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
    }

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
      <div className={`page-head targets-page-head${isHead ? '' : ' employee-targets-page-head'}`}>
        <h1>{t(isHead ? 'targets' : 'myTargets')}</h1>
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
          {isHead && (
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
          )}
          {isHead && (scope === 'officer' ? (
            <NameFilterCombobox
              items={officerOptions}
              value={officerId}
              onChange={setOfficerId}
              includeAll={false}
              placeholder={t('selectOfficer')}
              ariaLabel={t('selectOfficer')}
            />
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
                  {districtName(district, lang)}
                </option>
              ))}
            </select>
          ))}
          <KpiCodeCombobox
            items={kpiOptions}
            value={kpiFilter}
            onChange={setKpiFilter}
            includeAll
            allLabel={t('allKpis')}
            ariaLabel={t('searchKpiCode')}
          />
          {isHead && (
            <button className="btn btn-primary" onClick={saveTargets} disabled={saving || !selectionId}>
              {t('save')}
            </button>
          )}
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
              {visibleRows.map((row) => {
                const targetQuantity = Number(row.target_quantity || 0)
                const targetScore = isHead
                  ? Math.round(targetQuantity * row.points_per_unit * coeff * 100) / 100
                  : row.target_score
                return (
                  <tr key={row.catalog_id}>
                    <td><span className="code-chip">{row.code}</span></td>
                    <td>{kpiIndicator(row, lang)}</td>
                    <td className="muted">{row.unit}</td>
                    <td className="num">{row.points_per_unit}</td>
                    <td className="num target-cell">
                      {isHead ? (
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
                      ) : targetQuantity}
                    </td>
                    <td className="num strong">{row.mixed ? '—' : targetScore}</td>
                  </tr>
                )
              })}
              {!loading && visibleRows.length === 0 && (
                <tr>
                  <td colSpan={6} className="empty">
                    {targetRows.length === 0 ? t('noTargetsForMonth') : t('noKpiMatches')}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
