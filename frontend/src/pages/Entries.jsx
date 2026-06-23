import { useEffect, useMemo, useState } from 'react'
import api from '../api'
import { getApiErrorMessage } from '../apiError.js'
import { useAuth } from '../auth.jsx'
import StatusBadge from '../components/StatusBadge.jsx'
import EntryFormModal from '../components/EntryFormModal.jsx'
import { useI18n } from '../i18n.jsx'
import { useNotify } from '../notifications.jsx'

const NUMERIC_KEYS = ['quantity', 'points_per_unit', 'coeff', 'raw_score', 'final_score']
const currentMonth = () => new Date().toISOString().slice(0, 7)

function SummaryCard({ label, value, accent }) {
  return (
    <div className="stat-card" style={{ borderTopColor: accent }}>
      <div className="stat-value">{value}</div>
      <div className="stat-label">{label}</div>
    </div>
  )
}

// "2026-06" -> "2026-Q2"
const quarterOf = (ym) => {
  const [y, m] = ym.split('-')
  const q = Math.floor((parseInt(m, 10) - 1) / 3) + 1
  return `${y}-Q${q}`
}

function SortTh({ col, label, num, sort, onSort }) {
  const active = sort.key === col
  return (
    <th className={`sortable${num ? ' num' : ''}${active ? ' sorted' : ''}`} onClick={() => onSort(col)}>
      {label}
      <span className="sort-ind">{active ? (sort.dir === 'asc' ? '▲' : '▼') : '⇅'}</span>
    </th>
  )
}

export default function Entries() {
  const { t, fmtMonth, fmtQuarter } = useI18n()
  const { isHead } = useAuth()
  const { notify } = useNotify()
  const [entries, setEntries] = useState([])
  const [catalog, setCatalog] = useState([])
  const [statusFilter, setStatusFilter] = useState('')
  const [periodType, setPeriodType] = useState('') // '' | 'month' | 'quarter' | 'year'
  const [periodValue, setPeriodValue] = useState('')
  const [nameQuery, setNameQuery] = useState('')
  const [showModal, setShowModal] = useState(false)
  const [loading, setLoading] = useState(true)
  const [targetProgress, setTargetProgress] = useState(null)
  const [sort, setSort] = useState({ key: null, dir: 'asc' })

  const load = async () => {
    setLoading(true)
    const params = statusFilter ? { status: statusFilter } : {}
    const { data } = await api.get('/entries', { params })
    setEntries(data)
    setLoading(false)
  }

  useEffect(() => {
    api.get('/catalog').then((r) => setCatalog(r.data))
  }, [])

  useEffect(() => {
    load()
  }, [statusFilter])

  const toggleSort = (key) => {
    setSort((s) => (s.key === key ? { key, dir: s.dir === 'asc' ? 'desc' : 'asc' } : { key, dir: 'asc' }))
  }

  // Доступные периоды, выведенные из самих записей.
  const periodOptions = useMemo(() => {
    const months = new Set()
    const quarters = new Set()
    const years = new Set()
    for (const e of entries) {
      if (!e.month || !e.month.includes('-')) continue
      months.add(e.month)
      quarters.add(quarterOf(e.month))
      years.add(e.month.slice(0, 4))
    }
    return {
      month: [...months].sort().reverse(),
      quarter: [...quarters].sort().reverse(),
      year: [...years].sort().reverse(),
    }
  }, [entries])

  // При смене типа периода выбираем последний доступный период.
  useEffect(() => {
    if (!periodType) {
      setPeriodValue('')
      return
    }
    const opts = periodOptions[periodType] || []
    setPeriodValue((prev) => (opts.includes(prev) ? prev : opts[0] || ''))
  }, [periodType, periodOptions])

  const summaryMonth = periodType === 'month' && periodValue ? periodValue : currentMonth()

  useEffect(() => {
    if (isHead) return
    api.get('/target-progress', { params: { month: summaryMonth } })
      .then((r) => setTargetProgress(r.data))
      .catch(() => setTargetProgress(null))
  }, [isHead, summaryMonth])

  const periodLabel = (type, val) => {
    if (type === 'month') return fmtMonth(val)
    if (type === 'quarter') {
      const [y, q] = val.split('-Q')
      return fmtQuarter(y, q)
    }
    return val
  }

  const filteredEntries = useMemo(() => {
    const q = nameQuery.trim().toLowerCase()
    const periodActive = periodType && periodValue
    return entries.filter((e) => {
      if (q && !(e.officer_name || '').toLowerCase().includes(q)) return false
      if (!periodActive) return true
      if (!e.month || !e.month.includes('-')) return false
      if (periodType === 'month') return e.month === periodValue
      if (periodType === 'quarter') return quarterOf(e.month) === periodValue
      if (periodType === 'year') return e.month.slice(0, 4) === periodValue
      return true
    })
  }, [entries, periodType, periodValue, nameQuery])

  const sortedEntries = useMemo(() => {
    if (!sort.key) return filteredEntries
    const numeric = NUMERIC_KEYS.includes(sort.key)
    const factor = sort.dir === 'asc' ? 1 : -1
    return [...filteredEntries].sort((a, b) => {
      let av = a[sort.key]
      let bv = b[sort.key]
      if (numeric) {
        av = Number(av) || 0
        bv = Number(bv) || 0
      } else {
        av = (av ?? '').toString().toLowerCase()
        bv = (bv ?? '').toString().toLowerCase()
      }
      if (av < bv) return -1 * factor
      if (av > bv) return 1 * factor
      return 0
    })
  }, [filteredEntries, sort])

  const setStatus = async (id, status) => {
    let head_comment = ''
    if (status === 'rejected') {
      head_comment = window.prompt(t('headResolution')) || ''
    }
    try {
      const { data } = await api.patch(`/entries/${id}/status`, { status, head_comment })
      setEntries((prev) => prev.map((e) => (e.id === id ? data : e)))
      notify(t(status === 'confirmed' ? 'entryConfirmed' : 'entryRejected'))
    } catch (err) {
      notify(getApiErrorMessage(err, t, 'statusUpdateError'), 'error')
    }
  }

  return (
    <div>
      <div className="page-head">
        <h1>{isHead ? t('entries') : t('myEntries')}</h1>
        <div className="page-head-actions">
          {isHead && (
            <input
              className="search-input"
              type="search"
              placeholder={t('searchName')}
              value={nameQuery}
              onChange={(e) => setNameQuery(e.target.value)}
            />
          )}
          <select className="select" value={periodType} onChange={(e) => setPeriodType(e.target.value)}>
            <option value="">{t('periodAll')}</option>
            <option value="month">{t('byMonth')}</option>
            <option value="quarter">{t('byQuarter')}</option>
            <option value="year">{t('byYear')}</option>
          </select>
          {periodType && (
            <select className="select" value={periodValue} onChange={(e) => setPeriodValue(e.target.value)}>
              {(periodOptions[periodType] || []).map((v) => (
                <option key={v} value={v}>{periodLabel(periodType, v)}</option>
              ))}
            </select>
          )}
          <select className="select" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
            <option value="">{t('filterAll')}</option>
            <option value="pending">{t('pending')}</option>
            <option value="confirmed">{t('confirmed')}</option>
            <option value="rejected">{t('rejected')}</option>
          </select>
          {!isHead && (
            <button className="btn btn-primary" onClick={() => setShowModal(true)}>+ {t('addEntry')}</button>
          )}
        </div>
      </div>

      {!isHead && targetProgress && (
        <>
          <div className="summary-note">{t('summaryForMonth')} {fmtMonth(summaryMonth)}</div>
          <div className="stat-grid">
            <SummaryCard label={t('actualScore')} value={targetProgress.actual_score} accent="#1e4d8c" />
            <SummaryCard label={t('targetScore')} value={targetProgress.target_score || '—'} accent="#5b4b8a" />
            <SummaryCard
              label={t('achievement')}
              value={targetProgress.target_score ? `${targetProgress.achievement_percent}%` : '—'}
              accent="#2f7d4f"
            />
          </div>
        </>
      )}

      <div className="card table-card">
        <div className="table-scroll">
          <table className="data-table">
            <thead>
              <tr>
                <SortTh col="date" label={t('colDate')} sort={sort} onSort={toggleSort} />
                <SortTh col="month" label={t('colMonth')} sort={sort} onSort={toggleSort} />
                {isHead && <SortTh col="officer_name" label={t('colOfficer')} sort={sort} onSort={toggleSort} />}
                <SortTh col="code" label={t('colCode')} sort={sort} onSort={toggleSort} />
                <SortTh col="indicator" label={t('colIndicator')} sort={sort} onSort={toggleSort} />
                <SortTh col="block" label={t('colBlock')} sort={sort} onSort={toggleSort} />
                <SortTh col="npa" label={t('colNpa')} sort={sort} onSort={toggleSort} />
                <SortTh col="quantity" label={t('colQuantity')} num sort={sort} onSort={toggleSort} />
                <SortTh col="points_per_unit" label={t('colPointsPerUnit')} num sort={sort} onSort={toggleSort} />
                <SortTh col="coeff" label={t('colCoeff')} num sort={sort} onSort={toggleSort} />
                <SortTh col="raw_score" label={t('colRaw')} num sort={sort} onSort={toggleSort} />
                <SortTh col="final_score" label={t('colFinal')} num sort={sort} onSort={toggleSort} />
                <SortTh col="status" label={t('colStatus')} sort={sort} onSort={toggleSort} />
                <SortTh col="comments" label={t('colComments')} sort={sort} onSort={toggleSort} />
                {isHead && <th>{t('colActions')}</th>}
              </tr>
            </thead>
            <tbody>
              {sortedEntries.map((e) => (
                <tr key={e.id}>
                  <td>{e.date}</td>
                  <td className="nowrap">{fmtMonth(e.month)}</td>
                  {isHead && <td className="nowrap">{e.officer_name}</td>}
                  <td><span className="code-chip">{e.code}</span></td>
                  <td>{e.indicator}</td>
                  <td>{e.block}</td>
                  <td className="muted">{e.npa}</td>
                  <td className="num">{e.quantity}</td>
                  <td className="num">{e.points_per_unit}</td>
                  <td className="num">{e.coeff}</td>
                  <td className="num">{e.raw_score}</td>
                  <td className="num strong">{e.final_score}</td>
                  <td><StatusBadge status={e.status} /></td>
                  <td className="muted">
                    {e.comments}
                    {e.head_comment && <div className="head-comment">↳ {e.head_comment}</div>}
                  </td>
                  {isHead && (
                    <td>
                      <div className="row-actions">
                        <button
                          className="btn btn-sm btn-confirm"
                          disabled={e.status === 'confirmed'}
                          onClick={() => setStatus(e.id, 'confirmed')}
                        >
                          ✓
                        </button>
                        <button
                          className="btn btn-sm btn-reject"
                          disabled={e.status === 'rejected'}
                          onClick={() => setStatus(e.id, 'rejected')}
                        >
                          ✕
                        </button>
                      </div>
                    </td>
                  )}
                </tr>
              ))}
              {!loading && sortedEntries.length === 0 && (
                <tr><td colSpan={20} className="empty">{entries.length === 0 ? t('noEntries') : t('nothingFound')}</td></tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {showModal && (
        <EntryFormModal
          catalog={catalog}
          onClose={() => setShowModal(false)}
          onCreated={(entry) => {
            setEntries((prev) => [entry, ...prev])
            if (!isHead) {
              api.get('/target-progress', { params: { month: summaryMonth } })
                .then((r) => setTargetProgress(r.data))
                .catch(() => setTargetProgress(null))
            }
          }}
        />
      )}
    </div>
  )
}
