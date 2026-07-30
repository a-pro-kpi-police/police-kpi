import { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '../api'
import { useI18n } from '../i18n.jsx'

const kindMarks = {
  entry_pending: '↗',
  entry_confirmed: '✓',
  entry_rejected: '×',
  targets_updated: '◎',
}
const parseApiTimestamp = (value) => new Date(
  /(?:Z|[+-]\d{2}:\d{2})$/.test(value) ? value : `${value}Z`,
)

export default function NotificationCenter() {
  const { t, lang } = useI18n()
  const navigate = useNavigate()
  const rootRef = useRef(null)
  const [open, setOpen] = useState(false)
  const [items, setItems] = useState([])
  const [unreadCount, setUnreadCount] = useState(0)

  const load = useCallback(async () => {
    try {
      const { data } = await api.get('/notifications')
      setItems(data.items)
      setUnreadCount(data.unread_count)
    } catch {
      // The header remains usable if a refresh is temporarily unavailable.
    }
  }, [])

  useEffect(() => {
    load()
    const interval = window.setInterval(load, 15000)
    const refresh = () => load()
    window.addEventListener('notifications-updated', refresh)
    return () => {
      window.clearInterval(interval)
      window.removeEventListener('notifications-updated', refresh)
    }
  }, [load])

  useEffect(() => {
    if (!open) return undefined
    const closeOnOutsideClick = (event) => {
      if (!rootRef.current?.contains(event.target)) setOpen(false)
    }
    document.addEventListener('mousedown', closeOnOutsideClick)
    return () => document.removeEventListener('mousedown', closeOnOutsideClick)
  }, [open])

  const markAllRead = async () => {
    await api.post('/notifications/read-all')
    setItems((current) => current.map((item) => ({ ...item, is_read: true })))
    setUnreadCount(0)
  }

  const openNotification = async (item) => {
    if (!item.is_read) {
      await api.patch(`/notifications/${item.id}/read`)
      setItems((current) => current.map((entry) => (
        entry.id === item.id ? { ...entry, is_read: true } : entry
      )))
      setUnreadCount((count) => Math.max(0, count - 1))
    }
    setOpen(false)
    if (item.link) navigate(item.link)
  }

  const formatter = new Intl.DateTimeFormat(
    lang === 'kk' ? 'kk-KZ' : 'ru-RU',
    { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' },
  )

  return (
    <div className="notification-center" ref={rootRef}>
      <button
        type="button"
        className={`notification-button${open ? ' active' : ''}`}
        onClick={() => setOpen((value) => !value)}
        aria-label={t('notifications')}
        aria-expanded={open}
      >
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4" />
        </svg>
        {unreadCount > 0 && (
          <span className="notification-count">
            {unreadCount > 99 ? '99+' : unreadCount}
          </span>
        )}
      </button>

      {open && (
        <div className="notification-popover">
          <div className="notification-popover-head">
            <strong>{t('notifications')}</strong>
            {unreadCount > 0 && (
              <button type="button" onClick={markAllRead}>{t('markAllRead')}</button>
            )}
          </div>
          <div className="notification-list">
            {items.map((item) => (
              <button
                type="button"
                className={`notification-item${item.is_read ? '' : ' unread'}`}
                key={item.id}
                onClick={() => openNotification(item)}
              >
                <span className={`notification-kind notification-kind-${item.kind}`} aria-hidden="true">
                  {kindMarks[item.kind] || '•'}
                </span>
                <span className="notification-copy">
                  <strong>{item.title}</strong>
                  <span>{item.message}</span>
                  <time dateTime={item.created_at}>{formatter.format(parseApiTimestamp(item.created_at))}</time>
                </span>
              </button>
            ))}
            {!items.length && <div className="notification-empty">{t('noNotifications')}</div>}
          </div>
        </div>
      )}
    </div>
  )
}
