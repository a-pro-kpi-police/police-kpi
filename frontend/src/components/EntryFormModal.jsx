import { useEffect, useMemo, useState } from 'react'
import { useAuth } from '../auth.jsx'
import api from '../api'
import { getApiErrorMessage } from '../apiError.js'
import { useI18n } from '../i18n.jsx'
import { useNotify } from '../notifications.jsx'

export default function EntryFormModal({ catalog, onClose, onCreated }) {
  const { t } = useI18n()
  const { user } = useAuth()
  const { notify } = useNotify()
  const [catalogId, setCatalogId] = useState(catalog[0]?.id || '')
  const [quantity, setQuantity] = useState(1)
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10))
  const [comments, setComments] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [targetProgress, setTargetProgress] = useState(null)

  const coeff = user?.district?.coeff ?? 1.0
  const selected = useMemo(
    () => catalog.find((c) => c.id === Number(catalogId)),
    [catalog, catalogId]
  )
  const raw = selected ? selected.points_per_unit * Number(quantity || 0) : 0
  const final = Math.round(raw * coeff * 100) / 100
  const targetItem = targetProgress?.items?.find((item) => item.catalog_id === Number(catalogId))

  useEffect(() => {
    if (!date) return
    api.get('/target-progress', { params: { month: date.slice(0, 7) } })
      .then((r) => setTargetProgress(r.data))
      .catch(() => setTargetProgress(null))
  }, [date])

  const submit = async (e) => {
    e.preventDefault()
    setSaving(true)
    setError('')
    try {
      const { data } = await api.post('/entries', {
        catalog_id: Number(catalogId),
        quantity: Number(quantity),
        date,
        comments,
      })
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
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h2>{t('newEntry')}</h2>
        <form onSubmit={submit}>
          <label>{t('selectKpi')}</label>
          <select value={catalogId} onChange={(e) => setCatalogId(e.target.value)}>
            {catalog.map((c) => (
              <option key={c.id} value={c.id}>
                {c.code} — {c.indicator} ({c.points_per_unit} {t('points')}/{c.unit})
              </option>
            ))}
          </select>

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

          <div className="calc-box">
            <div><span>{t('colBlock')}</span><b>{selected?.block || '—'}</b></div>
            <div><span>{t('colNpa')}</span><b>{selected?.npa || '—'}</b></div>
            <div><span>{t('yourCoeff')}</span><b>{coeff}</b></div>
            <div><span>{t('monthlyTarget')}</span><b>{targetItem?.target_quantity ? `${targetItem.target_quantity} ${selected?.unit || ''}` : '—'}</b></div>
            <div><span>{t('confirmedSoFar')}</span><b>{targetItem ? `${targetItem.confirmed_quantity} ${selected?.unit || ''}` : '—'}</b></div>
            <div><span>{t('remainingQuantity')}</span><b>{targetItem?.target_quantity ? `${targetItem.remaining_quantity} ${selected?.unit || ''}` : '—'}</b></div>
            <div><span>{t('targetScore')}</span><b>{targetItem?.target_score || '—'}</b></div>
            <div><span>{t('computedRaw')}</span><b>{raw}</b></div>
            <div className="calc-final"><span>{t('computedFinal')}</span><b>{final}</b></div>
          </div>

          {error && <div className="error">{error}</div>}

          <div className="modal-actions">
            <button type="button" className="btn btn-ghost" onClick={onClose}>{t('cancel')}</button>
            <button type="submit" className="btn btn-primary" disabled={saving}>{t('save')}</button>
          </div>
        </form>
      </div>
    </div>
  )
}
