import { useEffect, useMemo, useState } from 'react'
import { useAuth } from '../auth.jsx'
import api from '../api'
import { getApiErrorMessage } from '../apiError.js'
import { districtName } from '../districtLocale.js'
import KpiCodeCombobox from './KpiCodeCombobox.jsx'
import NameFilterCombobox from './NameFilterCombobox.jsx'
import { useI18n } from '../i18n.jsx'
import { useNotify } from '../notifications.jsx'
import { kpiIndicator } from '../kpiLocale.js'

const MAX_ATTACHMENT_SIZE = 20 * 1024 * 1024
const MAX_ATTACHMENTS = 5
const ATTACHMENT_EXTENSIONS = [
  '.pdf', '.png', '.jpg', '.jpeg', '.webp', '.doc', '.docx',
  '.xls', '.xlsx', '.csv', '.txt', '.zip', '.mp4', '.webm', '.ogv',
]
const ATTACHMENT_ACCEPT = ATTACHMENT_EXTENSIONS.join(',')
const supportedAttachment = (file) => ATTACHMENT_EXTENSIONS.some(
  (extension) => file.name.toLowerCase().endsWith(extension)
)

export default function EntryFormModal({ catalog, officers, onClose, onCreated }) {
  const { t, lang } = useI18n()
  const { user, canManageEntries } = useAuth()
  const { notify } = useNotify()
  const [officerId, setOfficerId] = useState(canManageEntries ? officers[0]?.id || '' : user?.id || '')
  const [catalogId, setCatalogId] = useState(catalog[0]?.id || '')
  const [quantity, setQuantity] = useState(1)
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10))
  const [comments, setComments] = useState('')
  const [attachments, setAttachments] = useState([])
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [targetProgress, setTargetProgress] = useState(null)

  const officer = canManageEntries
    ? officers.find((item) => item.id === Number(officerId))
    : user
  const officerOptions = useMemo(
    () => officers.map((item) => ({
      id: item.id,
      label: item.full_name,
      description: [
        item.rank,
        districtName(item.district || item.district_name, lang),
      ].filter(Boolean).join(' · '),
    })),
    [lang, officers]
  )
  const coeff = officer?.district?.coeff ?? 1.0
  const selected = useMemo(
    () => catalog.find((c) => c.id === Number(catalogId)),
    [catalog, catalogId]
  )
  const raw = selected ? selected.points_per_unit * Number(quantity || 0) : 0
  const targetItem = targetProgress?.items?.find((item) => item.catalog_id === Number(catalogId))
  const remainingLimit = selected?.monthly_limit
    ? Math.max(selected.monthly_limit - Number(targetItem?.counted_quantity || 0), 0)
    : Number(quantity || 0)
  const countedForEntry = Math.min(Number(quantity || 0), remainingLimit)
  const capped = selected ? selected.points_per_unit * countedForEntry : 0
  const final = Math.round(capped * coeff * 100) / 100

  useEffect(() => {
    if (!date || (canManageEntries && !officerId)) {
      setTargetProgress(null)
      return
    }
    const params = { month: date.slice(0, 7) }
    if (canManageEntries) params.officer_id = Number(officerId)
    api.get('/target-progress', { params })
      .then((r) => setTargetProgress(r.data))
      .catch(() => setTargetProgress(null))
  }, [canManageEntries, date, officerId])

  const submit = async (e) => {
    e.preventDefault()
    if (!catalogId) return
    setSaving(true)
    setError('')
    try {
      const payload = {
        catalog_id: Number(catalogId),
        ...(canManageEntries ? { officer_id: Number(officerId) } : {}),
        quantity: Number(quantity),
        date,
        comments,
      }
      let response
      if (attachments.length) {
        const formData = new FormData()
        Object.entries(payload).forEach(([key, value]) => {
          formData.append(key, String(value))
        })
        attachments.forEach((attachment) => formData.append('attachments', attachment))
        response = await api.post('/entries/with-attachments', formData)
      } else {
        response = await api.post('/entries', payload)
      }
      const { data } = response
      onCreated(data)
      notify(t('entryCreated'))
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
      <div className="modal entry-form-modal" onClick={(e) => e.stopPropagation()}>
        <h2>{t('newEntry')}</h2>
        <form className="entry-form" onSubmit={submit}>
          <div className="entry-form-layout">
            <section className="entry-form-fields">
              {canManageEntries && (
                <>
                  <label>{t('selectOfficer')}</label>
                  <NameFilterCombobox
                    items={officerOptions}
                    value={officerId}
                    onChange={setOfficerId}
                    includeAll={false}
                    placeholder={t('selectOfficer')}
                    ariaLabel={t('selectOfficer')}
                    required
                  />
                </>
              )}

              <label>{t('selectKpi')}</label>
              <KpiCodeCombobox
                items={catalog}
                value={catalogId}
                onChange={setCatalogId}
                ariaLabel={t('selectKpi')}
                placeholder={t('searchKpiCodePlaceholder')}
                required
              />

              {selected && (
                <div className="selected-kpi-panel">
                  <span>{t('colIndicator')}</span>
                  <strong>{kpiIndicator(selected, lang)}</strong>
                </div>
              )}

              <div className="form-row">
                <div>
                  <label>{t('colDate')}</label>
                  <input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
                </div>
                <div>
                  <label>{t('quantity')}{selected ? ` (${selected.unit})` : ''}</label>
                  <input
                    type="number"
                    min="0.5"
                    step="0.5"
                    value={quantity}
                    onChange={(e) => setQuantity(e.target.value)}
                  />
                </div>
              </div>

              <label>{t('comments')}</label>
              <textarea rows={2} value={comments} onChange={(e) => setComments(e.target.value)} />

              <label htmlFor="entry-attachment">{t('attachment')}</label>
              <div className="file-picker">
                <input
                  id="entry-attachment"
                  className="file-picker-input"
                  type="file"
                  accept={ATTACHMENT_ACCEPT}
                  multiple
                  aria-label={t('attachment')}
                  onChange={(event) => {
                    const files = Array.from(event.target.files || [])
                    if (files.length > MAX_ATTACHMENTS) {
                      event.target.value = ''
                      setAttachments([])
                      setError(t('attachmentCountError'))
                      return
                    }
                    if (files.some((file) => file.size > MAX_ATTACHMENT_SIZE)) {
                      event.target.value = ''
                      setAttachments([])
                      setError(t('attachmentTooLargeError'))
                      return
                    }
                    if (files.some((file) => !supportedAttachment(file))) {
                      event.target.value = ''
                      setAttachments([])
                      setError(t('attachmentTypeError'))
                      return
                    }
                    setAttachments(files)
                    setError('')
                  }}
                />
                <label className="file-picker-button" htmlFor="entry-attachment">
                  {t('chooseFile')}
                </label>
                <span className="file-picker-name" title={attachments.map((file) => file.name).join(', ')}>
                  {attachments.length ? `${t('filesSelected')}: ${attachments.length}` : t('noFileSelected')}
                </span>
              </div>
              <div className="attachment-help">{t('attachmentHelp')}</div>
            </section>

            <aside className="calc-box entry-score-panel" aria-label={t('computedFinal')}>
              <div><span>{t('colBlock')}</span><b>{selected?.block || '—'}</b></div>
              <div><span>{t('colNpa')}</span><b>{selected?.npa || '—'}</b></div>
              <div>
                <span>{t('colSource')}</span>
                <b>
                  {selected?.source_url ? (
                    <a
                      className="source-link"
                      href={selected.source_url}
                      target="_blank"
                      rel="noreferrer"
                    >
                      {t('openSource')}
                    </a>
                  ) : '—'}
                </b>
              </div>
              <div><span>{t(canManageEntries ? 'employeeCoeff' : 'yourCoeff')}</span><b>{coeff}</b></div>
              <div><span>{t('monthlyLimit')}</span><b>{selected?.monthly_limit || '—'}</b></div>
              <div><span>{t('massIndicator')}</span><b>{selected?.is_mass ? t('yes') : t('no')}</b></div>
              <div><span>{t('criticalIndicator')}</span><b>{selected?.is_critical ? t('yes') : t('no')}</b></div>
              <div><span>{t('confirmedSoFar')}</span><b>{targetItem ? `${targetItem.confirmed_quantity} ${selected?.unit || ''}` : '—'}</b></div>
              <div><span>{t('countedSoFar')}</span><b>{targetItem ? `${targetItem.counted_quantity} ${selected?.unit || ''}` : '—'}</b></div>
              <div><span>{t('countedForEntry')}</span><b>{countedForEntry}</b></div>
              <div><span>{t('computedRaw')}</span><b>{raw}</b></div>
              <div><span>{t('cappedScore')}</span><b>{capped}</b></div>
              <div className="calc-final"><span>{t('preliminaryScore')}</span><b>{final}</b></div>
            </aside>
          </div>

          {error && <div className="error">{error}</div>}

          <div className="modal-actions">
            <button type="button" className="btn btn-ghost" onClick={onClose}>{t('cancel')}</button>
            <button
              type="submit"
              className="btn btn-primary"
              disabled={saving || !catalogId || (canManageEntries && !officerId)}
            >
              {t('save')}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
