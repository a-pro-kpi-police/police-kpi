import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '../api'
import { getApiErrorMessage } from '../apiError.js'
import { useAuth } from '../auth.jsx'
import StatusBadge from '../components/StatusBadge.jsx'
import EntryFormModal from '../components/EntryFormModal.jsx'
import NameFilterCombobox from '../components/NameFilterCombobox.jsx'
import ProfileHoverCard from '../components/ProfileHoverCard.jsx'
import { useI18n } from '../i18n.jsx'
import { kpiIndicator } from '../kpiLocale.js'
import { useNotify } from '../notifications.jsx'

const NUMERIC_KEYS = ['quantity', 'points_per_unit', 'coeff', 'raw_score', 'final_score']
const PAGE_SIZE = 20
const currentMonth = () => new Date().toISOString().slice(0, 7)
const formatFileSize = (bytes) => {
  if (!bytes) return '0 KB'
  if (bytes < 1024) return `${bytes} B`
  return `${Math.round((bytes / 1024) * 10) / 10} KB`
}

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

function RecordField({ label, children, wide = false }) {
  return (
    <div className={`entry-record-field${wide ? ' entry-record-field-wide' : ''}`}>
      <dt>{label}</dt>
      <dd>{children ?? '—'}</dd>
    </div>
  )
}

function MessageIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M5 5.5h14v10H9l-4 3v-13Z" />
    </svg>
  )
}

export default function Entries() {
  const { t, fmtMonth, fmtQuarter, lang } = useI18n()
  const { isHead } = useAuth()
  const { notify } = useNotify()
  const navigate = useNavigate()
  const [entries, setEntries] = useState([])
  const [catalog, setCatalog] = useState([])
  const [officers, setOfficers] = useState([])
  const [months, setMonths] = useState([])
  const [statusFilter, setStatusFilter] = useState('')
  const [periodType, setPeriodType] = useState('') // '' | 'month' | 'quarter' | 'year'
  const [periodValue, setPeriodValue] = useState('')
  const [officerFilter, setOfficerFilter] = useState('')
  const [showModal, setShowModal] = useState(false)
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [hasMore, setHasMore] = useState(true)
  const [targetProgress, setTargetProgress] = useState(null)
  const [sort, setSort] = useState({ key: 'date', dir: 'desc' })
  const loadMoreRef = useRef(null)
  const requestIdRef = useRef(0)
  const translationRef = useRef(t)

  useEffect(() => {
    translationRef.current = t
  }, [t])

  useEffect(() => {
    const requests = [
      api.get('/catalog'),
      api.get('/months'),
    ]
    if (isHead) requests.push(api.get('/officers'))
    Promise.all(requests).then(([catalogResponse, monthsResponse, officersResponse]) => {
      setCatalog(Array.isArray(catalogResponse.data) ? catalogResponse.data : [])
      setMonths(Array.isArray(monthsResponse.data) ? monthsResponse.data : [])
      if (officersResponse) {
        setOfficers(Array.isArray(officersResponse.data) ? officersResponse.data : [])
      }
    })
  }, [isHead])

  // Периоды загружаются отдельно, поэтому серверная пагинация не обрезает фильтры.
  const periodOptions = useMemo(() => {
    const quarters = new Set()
    const years = new Set()
    for (const month of months) {
      if (!month || !month.includes('-')) continue
      quarters.add(quarterOf(month))
      years.add(month.slice(0, 4))
    }
    return {
      month: [...months].sort().reverse(),
      quarter: [...quarters].sort().reverse(),
      year: [...years].sort().reverse(),
    }
  }, [months])
  const officerOptions = useMemo(
    () => officers.map((officer) => ({
      id: officer.id,
      label: officer.full_name,
      description: officer.rank,
    })),
    [officers]
  )
  const officerProfiles = useMemo(
    () => Object.fromEntries(officers.map((officer) => [officer.id, officer])),
    [officers],
  )

  // При смене типа периода выбираем последний доступный период.
  useEffect(() => {
    if (!periodType) {
      setPeriodValue('')
      return
    }
    const opts = periodOptions[periodType] || []
    setPeriodValue((prev) => (opts.includes(prev) ? prev : opts[0] || ''))
  }, [periodType, periodOptions])

  const entryQuery = useMemo(() => ({
    limit: PAGE_SIZE,
    status: statusFilter || undefined,
    officer_id: isHead && officerFilter ? Number(officerFilter) : undefined,
    period_type: periodType && periodValue ? periodType : undefined,
    period_value: periodType && periodValue ? periodValue : undefined,
    sort_key: sort.key,
    sort_dir: sort.dir,
  }), [
    isHead,
    officerFilter,
    periodType,
    periodValue,
    sort.dir,
    sort.key,
    statusFilter,
  ])

  const fetchEntryPage = useCallback(async (offset, append) => {
    const requestId = ++requestIdRef.current
    if (append) {
      setLoadingMore(true)
    } else {
      setLoading(true)
      setLoadingMore(false)
    }
    try {
      const { data } = await api.get('/entries', {
        params: { ...entryQuery, offset },
      })
      if (requestId !== requestIdRef.current) return
      setEntries((current) => (
        append
          ? [...current, ...data.filter(
            (entry) => !current.some((item) => item.id === entry.id),
          )]
          : data
      ))
      setHasMore(data.length === PAGE_SIZE)
    } catch (error) {
      if (requestId === requestIdRef.current) {
        setHasMore(false)
        notify(
          getApiErrorMessage(error, translationRef.current, 'entriesLoadError'),
          'error',
        )
      }
    } finally {
      if (requestId === requestIdRef.current) {
        setLoading(false)
        setLoadingMore(false)
      }
    }
  }, [entryQuery, notify])

  useEffect(() => {
    setEntries([])
    setHasMore(true)
    fetchEntryPage(0, false)
  }, [fetchEntryPage])

  const loadMore = useCallback(() => {
    if (loading || loadingMore || !hasMore) return
    fetchEntryPage(entries.length, true)
  }, [entries.length, fetchEntryPage, hasMore, loading, loadingMore])

  useEffect(() => {
    const node = loadMoreRef.current
    if (!node || loading || loadingMore || !hasMore) return undefined
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) loadMore()
      },
      { rootMargin: '280px 0px' },
    )
    observer.observe(node)
    return () => observer.disconnect()
  }, [hasMore, loadMore, loading, loadingMore])

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
    const periodActive = periodType && periodValue
    return entries.filter((e) => {
      if (officerFilter && String(e.officer_id) !== String(officerFilter)) return false
      if (!periodActive) return true
      if (!e.month || !e.month.includes('-')) return false
      if (periodType === 'month') return e.month === periodValue
      if (periodType === 'quarter') return quarterOf(e.month) === periodValue
      if (periodType === 'year') return e.month.slice(0, 4) === periodValue
      return true
    })
  }, [entries, officerFilter, periodType, periodValue])

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

  const downloadAttachment = async (entry) => {
    try {
      const { data } = await api.get(`/entries/${entry.id}/attachment`, {
        responseType: 'blob',
      })
      const url = URL.createObjectURL(data)
      const link = document.createElement('a')
      link.href = url
      link.download = entry.attachment_name
      document.body.appendChild(link)
      link.click()
      link.remove()
      window.setTimeout(() => URL.revokeObjectURL(url), 1000)
    } catch {
      notify(t('attachmentDownloadError'), 'error')
    }
  }

  const filterByOfficer = (officerId) => {
    setOfficerFilter(String(officerId))
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const discussEntry = (entry) => {
    const params = new URLSearchParams({ entry: String(entry.id) })
    if (isHead) params.set('contact', String(entry.officer_id))
    navigate(`/messenger?${params.toString()}`)
  }

  return (
    <div>
      <div className="page-head entries-page-head">
        <h1>{isHead ? t('entries') : t('myEntries')}</h1>
        <div className={`page-head-actions entries-filters${periodType ? ' has-period-value' : ''}`}>
          {isHead && (
            <NameFilterCombobox
              items={officerOptions}
              value={officerFilter}
              onChange={setOfficerFilter}
            />
          )}
          <select className="select period-type-select" value={periodType} onChange={(e) => setPeriodType(e.target.value)}>
            <option value="">{t('periodAll')}</option>
            <option value="month">{t('byMonth')}</option>
            <option value="quarter">{t('byQuarter')}</option>
            <option value="year">{t('byYear')}</option>
          </select>
          {periodType && (
            <select className="select period-value-select" value={periodValue} onChange={(e) => setPeriodValue(e.target.value)}>
              {(periodOptions[periodType] || []).map((v) => (
                <option key={v} value={v}>{periodLabel(periodType, v)}</option>
              ))}
            </select>
          )}
          <select className="select status-select" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
            <option value="">{t('filterAll')}</option>
            <option value="pending">{t('pending')}</option>
            <option value="confirmed">{t('confirmed')}</option>
            <option value="rejected">{t('rejected')}</option>
          </select>
          <button
            className="btn btn-primary"
            disabled={!catalog.length || (isHead && !officers.length)}
            onClick={() => setShowModal(true)}
          >
            + {t('addEntry')}
          </button>
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

      <div className="entries-records-toolbar">
        <label htmlFor="entry-sort">{t('sortBy')}</label>
        <select
          id="entry-sort"
          className="select"
          value={sort.key}
          onChange={(event) => setSort((current) => ({ ...current, key: event.target.value }))}
        >
          <option value="date">{t('colDate')}</option>
          <option value="month">{t('colMonth')}</option>
          {isHead && <option value="officer_name">{t('colOfficer')}</option>}
          <option value="code">{t('colCode')}</option>
          <option value="final_score">{t('colFinal')}</option>
          <option value="status">{t('colStatus')}</option>
        </select>
        <button
          type="button"
          className="sort-direction"
          onClick={() => setSort((current) => ({
            ...current,
            dir: current.dir === 'asc' ? 'desc' : 'asc',
          }))}
          aria-label={t(sort.dir === 'asc' ? 'sortAscending' : 'sortDescending')}
          title={t(sort.dir === 'asc' ? 'sortAscending' : 'sortDescending')}
        >
          {sort.dir === 'asc' ? '↑' : '↓'}
        </button>
      </div>

      <div className="entry-record-grid">
        {sortedEntries.map((e) => (
          <article className="entry-record-card" key={e.id}>
            <header className="entry-record-header">
              <div className="entry-record-heading">
                <div className="entry-record-kicker">
                  <span className="code-chip">{e.code}</span>
                  <span>{fmtMonth(e.month)}</span>
                  <span aria-hidden="true">·</span>
                  <time dateTime={e.date}>{e.date}</time>
                </div>
                {isHead && (
                  <ProfileHoverCard profile={{
                    id: e.officer_id,
                    name: e.officer_name,
                    rank: e.rank,
                    district: e.district_name,
                    efficiencyRank: officerProfiles[e.officer_id]?.efficiency_rank,
                    efficiencyTotal: officerProfiles[e.officer_id]?.efficiency_total,
                    totalScore: officerProfiles[e.officer_id]?.efficiency_score,
                  }}>
                    <button
                      type="button"
                      className="entry-record-officer"
                      onClick={() => filterByOfficer(e.officer_id)}
                      aria-label={`${t('filterOfficerRecords')}: ${e.officer_name}`}
                      title={t('filterOfficerRecords')}
                    >
                      {e.officer_name}
                    </button>
                  </ProfileHoverCard>
                )}
                <h2>{kpiIndicator(e, lang)}</h2>
              </div>
              <StatusBadge status={e.status} />
            </header>

            <dl className="entry-record-metrics">
              <RecordField label={t('colQuantity')}>{e.quantity}</RecordField>
              <RecordField label={t('colCountedQuantity')}>{e.counted_quantity}</RecordField>
              <RecordField label={t('colFinal')}>
                <strong>{e.final_score}</strong>
              </RecordField>
            </dl>

            <dl className="entry-record-details">
              <RecordField label={t('colBlock')}>{e.block}</RecordField>
              <RecordField label={t('colNpa')}>{e.npa}</RecordField>
              <RecordField label={t('colPointsPerUnit')}>{e.points_per_unit}</RecordField>
              <RecordField label={t('monthlyLimit')}>{e.monthly_limit}</RecordField>
              <RecordField label={t('colRaw')}>{e.raw_score}</RecordField>
              <RecordField label={t('cappedScore')}>{e.capped_score}</RecordField>
              <RecordField label={t('massIndicator')}>{e.is_mass ? t('yes') : t('no')}</RecordField>
              <RecordField label={t('criticalIndicator')}>{e.is_critical ? t('yes') : t('no')}</RecordField>
              <RecordField label={t('antiCoefficient')}>{e.anti_coeff}</RecordField>
              <RecordField label={t('colCoeff')}>{e.coeff}</RecordField>
              <RecordField label={t('colSource')} wide>
                {e.source_url ? (
                  <a
                    className="source-link"
                    href={e.source_url}
                    target="_blank"
                    rel="noreferrer"
                  >
                    {t('openSource')}
                  </a>
                ) : '—'}
              </RecordField>
            </dl>

            {(e.comments || e.attachment_name || e.head_comment) && (
              <div className="entry-record-notes">
                {e.comments && (
                  <div>
                    <span>{t('colComments')}</span>
                    <p>{e.comments}</p>
                  </div>
                )}
                {e.attachment_name && (
                  <button
                    type="button"
                    className="attachment-link"
                    onClick={() => downloadAttachment(e)}
                    title={e.attachment_name}
                    aria-label={`${t('downloadAttachment')}: ${e.attachment_name}`}
                  >
                    <span aria-hidden="true">📎</span>
                    <span className="attachment-name">{e.attachment_name}</span>
                    <span className="attachment-size">({formatFileSize(e.attachment_size)})</span>
                  </button>
                )}
                {e.head_comment && <div className="head-comment">↳ {e.head_comment}</div>}
              </div>
            )}

            <footer className="entry-record-actions">
              <button
                type="button"
                className="btn btn-sm btn-ghost entry-record-message"
                onClick={() => discussEntry(e)}
                aria-label={t('discussThisKpi')}
                title={t('discussThisKpi')}
              >
                <MessageIcon />
                {t('sendMessage')}
              </button>
              {isHead && (
                <div className="entry-record-review-actions">
                  <button
                    className="btn btn-sm btn-confirm"
                    disabled={e.status === 'confirmed'}
                    onClick={() => setStatus(e.id, 'confirmed')}
                    aria-label={t('confirm')}
                    title={t('confirm')}
                  >
                    ✓ {t('confirm')}
                  </button>
                  <button
                    className="btn btn-sm btn-reject"
                    disabled={e.status === 'rejected'}
                    onClick={() => setStatus(e.id, 'rejected')}
                    aria-label={t('reject')}
                    title={t('reject')}
                  >
                    ✕ {t('reject')}
                  </button>
                </div>
              )}
            </footer>
          </article>
        ))}
      </div>

      <div
        ref={loadMoreRef}
        className="entries-load-sentinel"
        role={loading || loadingMore ? 'status' : undefined}
        aria-live="polite"
      >
        {(loading || loadingMore) && t('loadingEntries')}
      </div>

      {!loading && sortedEntries.length === 0 && (
        <div className="card entries-empty">
          {entries.length === 0 ? t('noEntries') : t('nothingFound')}
        </div>
      )}

      {showModal && (
        <EntryFormModal
          catalog={catalog}
          officers={officers}
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
