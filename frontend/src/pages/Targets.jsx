import { useEffect, useMemo, useState } from 'react'
import api from '../api'
import { getApiErrorMessage } from '../apiError.js'
import KpiCodeCombobox from '../components/KpiCodeCombobox.jsx'
import { useI18n } from '../i18n.jsx'
import { kpiIndicator } from '../kpiLocale.js'
import { useNotify } from '../notifications.jsx'

const isTargetKpi = (item) => Number(item.points_per_unit) >= 0

export default function Targets() {
  const { t, lang } = useI18n()
  const { notify } = useNotify()
  const [rows, setRows] = useState([])
  const [kpiFilter, setKpiFilter] = useState('')
  const [loading, setLoading] = useState(true)

  const targetRows = useMemo(
    () => rows.filter(isTargetKpi),
    [rows]
  )
  const kpiOptions = useMemo(
    () => targetRows.map((row) => ({
      id: row.id,
      code: row.code,
      indicator: row.indicator,
      points_per_unit: row.points_per_unit,
      unit: row.unit,
    })),
    [targetRows]
  )
  const visibleRows = useMemo(
    () => targetRows.filter((row) => !kpiFilter || String(row.id) === String(kpiFilter)),
    [kpiFilter, targetRows]
  )

  useEffect(() => {
    let active = true
    setLoading(true)
    api.get('/catalog')
      .then(({ data }) => {
        if (!active) return
        const items = Array.isArray(data) ? data : []
        setRows(items)
        setKpiFilter((current) => (
          current && !items.some((row) => String(row.id) === String(current))
            ? ''
            : current
        ))
      })
      .catch((err) => {
        if (!active) return
        notify(getApiErrorMessage(err, t, 'targetsLoadError'), 'error')
        setRows([])
      })
      .finally(() => {
        if (active) setLoading(false)
      })

    return () => {
      active = false
    }
  }, [])

  return (
    <div>
      <div className="page-head targets-page-head">
        <h1>{t('targets')}</h1>
        <div className="page-head-actions">
          <KpiCodeCombobox
            items={kpiOptions}
            value={kpiFilter}
            onChange={setKpiFilter}
            includeAll
            allLabel={t('allKpis')}
            ariaLabel={t('searchKpiCode')}
          />
        </div>
      </div>

      <div className="card table-card">
        <div className="table-scroll">
          <table className="data-table">
            <thead>
              <tr>
                <th>{t('colCode')}</th>
                <th>{t('colIndicator')}</th>
                <th>{t('colBlock')}</th>
                <th>{t('colUnit')}</th>
                <th className="num">{t('colPointsPerUnit')}</th>
                <th className="num">{t('monthlyLimit')}</th>
                <th>{t('massIndicator')}</th>
                <th>{t('criticalIndicator')}</th>
                <th>{t('colSource')}</th>
              </tr>
            </thead>
            <tbody>
              {loading && (
                <tr>
                  <td colSpan={9} className="empty">{t('loadingTargets')}</td>
                </tr>
              )}
              {!loading && visibleRows.map((row) => (
                <tr key={row.id}>
                  <td><span className="code-chip">{row.code}</span></td>
                  <td>{kpiIndicator(row, lang)}</td>
                  <td>{row.block}</td>
                  <td className="muted">{row.unit}</td>
                  <td className="num">{row.points_per_unit}</td>
                  <td className="num">{row.monthly_limit || '—'}</td>
                  <td>{row.is_mass ? t('yes') : t('no')}</td>
                  <td>{row.is_critical ? t('yes') : t('no')}</td>
                  <td>
                    {row.source_url ? (
                      <a href={row.source_url} target="_blank" rel="noreferrer">
                        {t('openSource')}
                      </a>
                    ) : '—'}
                  </td>
                </tr>
              ))}
              {!loading && visibleRows.length === 0 && (
                <tr>
                  <td colSpan={9} className="empty">
                    {targetRows.length === 0 ? t('nothingFound') : t('noKpiMatches')}
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
