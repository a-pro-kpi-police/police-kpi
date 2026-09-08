import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '../api'
import { getApiErrorMessage } from '../apiError.js'
import { useAuth } from '../auth.jsx'
import {
  defaultEntryStatusFilter,
  entryMatchesStatus,
  reconcileEntryStatus,
} from '../entryReview.js'
import StatusBadge from '../components/StatusBadge.jsx'
import EntryFormModal from '../components/EntryFormModal.jsx'
import NameFilterCombobox from '../components/NameFilterCombobox.jsx'
import ProfileHoverCard from '../components/ProfileHoverCard.jsx'
import { useI18n } from '../i18n.jsx'
import { kpiIndicator } from '../kpiLocale.js'
import { useNotify } from '../notifications.jsx'
import {
  attachmentBlob, attachmentPath, entryAttachments, imageAttachmentType,
  isImageAttachment, isVideoAttachment,
} from '../attachments.js'

const NUMERIC_KEYS = ['quantity', 'points_per_unit', 'coeff', 'raw_score', 'final_score']
const PAGE_SIZE = 20
const MAX_ATTACHMENT_SIZE = 20 * 1024 * 1024
const MAX_ATTACHMENTS = 5
const ATTACHMENT_ACCEPT = '.pdf,.png,.jpg,.jpeg,.webp,.doc,.docx,.xls,.xlsx,.csv,.txt,.zip,.mp4,.webm,.ogv'
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

function AttachmentPreview({ preview, onClose, onLoadError, t }) {
  const closeButtonRef = useRef(null)

  useEffect(() => {
    const handleKeyDown = (event) => {
      if (event.key === 'Escape') onClose()
      if (event.key === 'Tab') {
        event.preventDefault()
        closeButtonRef.current?.focus()
      }
    }
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.body.style.overflow = previousOverflow
      preview.returnFocus?.focus()
    }
  }, [onClose, preview.returnFocus])

  return (
    <div
      className="modal-overlay attachment-preview-overlay"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose()
      }}
    >
      <section
        className="attachment-preview-modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="attachment-preview-title"
      >
        <header>
          <h2 id="attachment-preview-title">{preview.name}</h2>
          <button
            ref={closeButtonRef}
            type="button"
            className="attachment-preview-close"
            onClick={onClose}
            aria-label={t('closeAttachmentPreview')}
            title={t('closeAttachmentPreview')}
            autoFocus
          >
            ×
          </button>
        </header>
        <div className="attachment-preview-canvas">
          {preview.kind === 'video' ? (
            <video
              src={preview.url}
              controls
              autoPlay
              preload="metadata"
              onError={onLoadError}
            >
              {t('videoPlaybackUnsupported')}
            </video>
          ) : (
            <img
              src={preview.url}
              alt={`${t('imagePreview')}: ${preview.name}`}
              onError={onLoadError}
            />
          )}
        </div>
      </section>
    </div>
  )
}

export default function Entries() {
  const { t, fmtMonth, fmtQuarter, lang } = useI18n()
  const { user, canManageEntries } = useAuth()
  const { notify } = useNotify()
  const navigate = useNavigate()
  const [entries, setEntries] = useState([])
  const [catalog, setCatalog] = useState([])
  const [officers, setOfficers] = useState([])
  const [months, setMonths] = useState([])
  const [statusFilter, setStatusFilter] = useState(
    () => defaultEntryStatusFilter(canManageEntries),
  )
  const [periodType, setPeriodType] = useState('') // '' | 'month' | 'quarter' | 'year'
  const [periodValue, setPeriodValue] = useState('')
  const [officerFilter, setOfficerFilter] = useState('')
  const [showModal, setShowModal] = useState(false)
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [hasMore, setHasMore] = useState(true)
  const [targetProgress, setTargetProgress] = useState(null)
  const [attachmentPreview, setAttachmentPreview] = useState(null)
  const [previewingAttachmentKey, setPreviewingAttachmentKey] = useState(null)
  const [uploadingEntryId, setUploadingEntryId] = useState(null)
  const [updatingEntryIds, setUpdatingEntryIds] = useState(() => new Set())
  const [sort, setSort] = useState({ key: 'date', dir: 'desc' })
  const loadMoreRef = useRef(null)
  const requestIdRef = useRef(0)
  const entryQueryRef = useRef(null)
  const previewRequestRef = useRef(null)
  const statusUpdateIdsRef = useRef(new Set())
  const translationRef = useRef(t)

  useEffect(() => {
    translationRef.current = t
  }, [t])

  useEffect(() => {
    const requests = [
      api.get('/catalog'),
      api.get('/months'),
    ]
    if (canManageEntries) requests.push(api.get('/officers'))
    Promise.all(requests).then(([catalogResponse, monthsResponse, officersResponse]) => {
      setCatalog(Array.isArray(catalogResponse.data) ? catalogResponse.data : [])
      setMonths(Array.isArray(monthsResponse.data) ? monthsResponse.data : [])
      if (officersResponse) {
        setOfficers(Array.isArray(officersResponse.data) ? officersResponse.data : [])
      }
    })
  }, [canManageEntries])

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
    officer_id: canManageEntries && officerFilter ? Number(officerFilter) : undefined,
    period_type: periodType && periodValue ? periodType : undefined,
    period_value: periodType && periodValue ? periodValue : undefined,
    sort_key: sort.key,
    sort_dir: sort.dir,
  }), [
    canManageEntries,
    officerFilter,
    periodType,
    periodValue,
    sort.dir,
    sort.key,
    statusFilter,
  ])
  entryQueryRef.current = entryQuery

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
        params: { ...entryQueryRef.current, offset },
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
  }, [notify])

  useEffect(() => {
    setHasMore(true)
    fetchEntryPage(0, false)
  }, [entryQuery, fetchEntryPage])

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
    if (canManageEntries) return
    api.get('/target-progress', { params: { month: summaryMonth } })
      .then((r) => setTargetProgress(r.data))
      .catch(() => setTargetProgress(null))
  }, [canManageEntries, summaryMonth])

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
      if (!entryMatchesStatus(e, statusFilter)) return false
      if (officerFilter && String(e.officer_id) !== String(officerFilter)) return false
      if (!periodActive) return true
      if (!e.month || !e.month.includes('-')) return false
      if (periodType === 'month') return e.month === periodValue
      if (periodType === 'quarter') return quarterOf(e.month) === periodValue
      if (periodType === 'year') return e.month.slice(0, 4) === periodValue
      return true
    })
  }, [entries, officerFilter, periodType, periodValue, statusFilter])

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
    if (statusUpdateIdsRef.current.has(id)) return

    let head_comment = ''
    if (status === 'rejected') {
      head_comment = window.prompt(t('headResolution')) || ''
    }

    statusUpdateIdsRef.current.add(id)
    setUpdatingEntryIds(new Set(statusUpdateIdsRef.current))
    try {
      const { data } = await api.patch(`/entries/${id}/status`, { status, head_comment })
      const activeStatusFilter = entryQueryRef.current?.status || ''
      setEntries((prev) => reconcileEntryStatus(prev, data, activeStatusFilter))
      notify(t(status === 'confirmed' ? 'entryConfirmed' : 'entryRejected'))
      fetchEntryPage(0, false)
    } catch (err) {
      notify(getApiErrorMessage(err, t, 'statusUpdateError'), 'error')
    } finally {
      statusUpdateIdsRef.current.delete(id)
      setUpdatingEntryIds(new Set(statusUpdateIdsRef.current))
    }
  }

  const downloadAttachment = async (entry, attachment) => {
    try {
      const { data } = await api.get(attachmentPath(entry.id, attachment), {
        responseType: 'blob',
      })
      const url = URL.createObjectURL(data)
      const link = document.createElement('a')
      link.href = url
      link.download = attachment.name
      document.body.appendChild(link)
      link.click()
      link.remove()
      window.setTimeout(() => URL.revokeObjectURL(url), 1000)
    } catch {
      notify(t('attachmentDownloadError'), 'error')
    }
  }

  const closeAttachmentPreview = useCallback(() => {
    setAttachmentPreview(null)
  }, [])

  useEffect(() => {
    const url = attachmentPreview?.url
    return () => {
      if (url && attachmentPreview?.kind === 'image') URL.revokeObjectURL(url)
    }
  }, [attachmentPreview?.kind, attachmentPreview?.url])

  useEffect(() => () => {
    const controller = previewRequestRef.current
    previewRequestRef.current = null
    controller?.abort()
  }, [])

  const previewAttachment = async (entry, attachment) => {
    previewRequestRef.current?.abort()
    const controller = new AbortController()
    previewRequestRef.current = controller
    const returnFocus = document.activeElement instanceof HTMLElement
      ? document.activeElement
      : null
    const attachmentKey = `${entry.id}-${attachment.id || 'legacy'}`
    setPreviewingAttachmentKey(attachmentKey)
    try {
      const { data } = await api.get(attachmentPath(entry.id, attachment), {
        responseType: 'blob',
        signal: controller.signal,
      })
      if (controller.signal.aborted) return
      const contentType = imageAttachmentType(attachment)
      const url = URL.createObjectURL(attachmentBlob(data, contentType))
      setAttachmentPreview({ url, name: attachment.name, kind: 'image', returnFocus })
    } catch {
      if (!controller.signal.aborted) notify(t('attachmentPreviewError'), 'error')
    } finally {
      if (previewRequestRef.current === controller) {
        previewRequestRef.current = null
        setPreviewingAttachmentKey(null)
      }
    }
  }

  const playAttachment = async (entry, attachment) => {
    const returnFocus = document.activeElement instanceof HTMLElement
      ? document.activeElement
      : null
    const attachmentKey = `${entry.id}-${attachment.id}`
    setPreviewingAttachmentKey(attachmentKey)
    try {
      const { data } = await api.get(
        `/entries/${entry.id}/attachments/${attachment.id}/playback-url`
      )
      setAttachmentPreview({ url: data.url, name: attachment.name, kind: 'video', returnFocus })
    } catch {
      notify(t('videoPlaybackError'), 'error')
    } finally {
      setPreviewingAttachmentKey(null)
    }
  }

  const addAttachments = async (entry, files) => {
    if (!files.length) return
    if (files.length > MAX_ATTACHMENTS) {
      notify(t('attachmentCountError'), 'error')
      return
    }
    if (files.some((file) => file.size > MAX_ATTACHMENT_SIZE)) {
      notify(t('attachmentTooLargeError'), 'error')
      return
    }
    const formData = new FormData()
    files.forEach((file) => formData.append('attachments', file))
    setUploadingEntryId(entry.id)
    try {
      const { data } = await api.post(`/entries/${entry.id}/attachments`, formData)
      setEntries((previous) => previous.map((item) => (item.id === entry.id ? data : item)))
      notify(t('attachmentsAdded'))
    } catch (err) {
      notify(getApiErrorMessage(err, t, 'attachmentUploadError'), 'error')
    } finally {
      setUploadingEntryId(null)
    }
  }

  const handleAttachmentPreviewError = useCallback(() => {
    const errorKey = attachmentPreview?.kind === 'video'
      ? 'videoPlaybackError'
      : 'attachmentPreviewError'
    closeAttachmentPreview()
    notify(t(errorKey), 'error')
  }, [attachmentPreview?.kind, closeAttachmentPreview, notify, t])

  const filterByOfficer = (officerId) => {
    setOfficerFilter(String(officerId))
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const discussEntry = (entry) => {
    const params = new URLSearchParams({ entry: String(entry.id) })
    if (user?.role === 'head') params.set('contact', String(entry.officer_id))
    navigate(`/messenger?${params.toString()}`)
  }

  const deleteEntry = async (entry) => {
    if (!window.confirm(t('confirmDeleteEntry'))) return
    try {
      await api.delete(`/entries/${entry.id}`)
      setEntries((prev) => prev.filter((item) => item.id !== entry.id))
      notify(t('entryDeleted'))
    } catch (err) {
      notify(getApiErrorMessage(err, t, 'entryDeleteError'), 'error')
    }
  }

  return (
    <div>
      <div className="page-head entries-page-head">
        <h1>{canManageEntries ? t('entries') : t('myEntries')}</h1>
        <div className={`page-head-actions entries-filters${periodType ? ' has-period-value' : ''}`}>
          {canManageEntries && (
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
            disabled={!catalog.length || (canManageEntries && !officers.length)}
            onClick={() => setShowModal(true)}
          >
            + {t('addEntry')}
          </button>
        </div>
      </div>

      {!canManageEntries && targetProgress && (
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
          {canManageEntries && <option value="officer_name">{t('colOfficer')}</option>}
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
        {sortedEntries.map((e) => {
          const attachments = entryAttachments(e)
          const statusUpdating = updatingEntryIds.has(e.id)
          return (
          <article className="entry-record-card" key={e.id}>
            <header className="entry-record-header">
              <div className="entry-record-heading">
                <div className="entry-record-kicker">
                  <span className="code-chip">{e.code}</span>
                  <span>{fmtMonth(e.month)}</span>
                  <span aria-hidden="true">·</span>
                  <time dateTime={e.date}>{e.date}</time>
                </div>
                {canManageEntries && (
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

            {(e.comments || attachments.length || e.head_comment) && (
              <div className="entry-record-notes">
                {e.comments && (
                  <div>
                    <span>{t('colComments')}</span>
                    <p>{e.comments}</p>
                  </div>
                )}
                {attachments.length > 0 && (
                  <div className="attachment-list">
                    {attachments.map((attachment) => {
                      const attachmentKey = `${e.id}-${attachment.id || 'legacy'}`
                      return (
                        <div className="attachment-details" key={attachmentKey}>
                          <button
                            type="button"
                            className="attachment-link attachment-download-link"
                            onClick={() => downloadAttachment(e, attachment)}
                            title={attachment.name}
                            aria-label={`${t('downloadAttachment')}: ${attachment.name}`}
                          >
                            <span aria-hidden="true">📎</span>
                            <span className="attachment-name">{attachment.name}</span>
                            <span className="attachment-size">({formatFileSize(attachment.size)})</span>
                          </button>
                          {isImageAttachment(attachment) && (
                            <button
                              type="button"
                              className="attachment-link attachment-preview-link"
                              onClick={() => previewAttachment(e, attachment)}
                              disabled={previewingAttachmentKey === attachmentKey}
                              aria-label={`${t('previewAttachment')}: ${attachment.name}`}
                            >
                              {previewingAttachmentKey === attachmentKey ? t('loadingPreview') : t('previewAttachment')}
                            </button>
                          )}
                          {isVideoAttachment(attachment) && attachment.id && (
                            <button
                              type="button"
                              className="attachment-link attachment-preview-link"
                              onClick={() => playAttachment(e, attachment)}
                              disabled={previewingAttachmentKey === attachmentKey}
                              aria-label={`${t('playVideo')}: ${attachment.name}`}
                            >
                              {previewingAttachmentKey === attachmentKey ? t('loadingPreview') : t('playVideo')}
                            </button>
                          )}
                        </div>
                      )
                    })}
                  </div>
                )}
                {e.head_comment && <div className="head-comment">↳ {e.head_comment}</div>}
              </div>
            )}

            <footer className="entry-record-actions">
              {user?.role !== 'admin' && (
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
              )}
              {canManageEntries && (
                <div className="entry-record-review-actions">
                  <button
                    className="btn btn-sm btn-confirm"
                    disabled={statusUpdating || e.status === 'confirmed'}
                    onClick={() => setStatus(e.id, 'confirmed')}
                    aria-label={t('confirm')}
                    title={t('confirm')}
                  >
                    ✓ {t('confirm')}
                  </button>
                  <button
                    className="btn btn-sm btn-reject"
                    disabled={statusUpdating || e.status === 'rejected'}
                    onClick={() => setStatus(e.id, 'rejected')}
                    aria-label={t('reject')}
                    title={t('reject')}
                  >
                    ✕ {t('reject')}
                  </button>
                  <button
                    className="btn btn-sm btn-delete"
                    onClick={() => deleteEntry(e)}
                    aria-label={t('deleteEntry')}
                    title={t('deleteEntry')}
                  >
                    {t('delete')}
                  </button>
                </div>
              )}
              <label className={`btn btn-sm btn-ghost entry-attachment-add${uploadingEntryId === e.id ? ' disabled' : ''}`}>
                {uploadingEntryId === e.id ? t('uploadingAttachments') : t('addAttachments')}
                <input
                  type="file"
                  accept={ATTACHMENT_ACCEPT}
                  multiple
                  disabled={uploadingEntryId === e.id}
                  onChange={(event) => {
                    const files = Array.from(event.target.files || [])
                    event.target.value = ''
                    addAttachments(e, files)
                  }}
                />
              </label>
            </footer>
          </article>
          )
        })}
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
            if (!canManageEntries) {
              api.get('/target-progress', { params: { month: summaryMonth } })
                .then((r) => setTargetProgress(r.data))
                .catch(() => setTargetProgress(null))
            }
          }}
        />
      )}

      {attachmentPreview && (
        <AttachmentPreview
          preview={attachmentPreview}
          onClose={closeAttachmentPreview}
          onLoadError={handleAttachmentPreviewError}
          t={t}
        />
      )}
    </div>
  )
}
