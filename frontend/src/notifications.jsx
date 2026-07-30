import { createContext, useCallback, useContext, useMemo, useState } from 'react'
import { useI18n } from './i18n.jsx'

const NotificationContext = createContext(null)

let nextId = 1
const EXIT_ANIMATION_MS = 180
const VISIBLE_DURATION_MS = 3500

export function NotificationProvider({ children }) {
  const { t } = useI18n()
  const [items, setItems] = useState([])

  const remove = useCallback((id) => {
    setItems((prev) => prev.map((item) => (
      item.id === id ? { ...item, closing: true } : item
    )))
    window.setTimeout(() => {
      setItems((prev) => prev.filter((item) => item.id !== id))
    }, EXIT_ANIMATION_MS)
  }, [])

  const notify = useCallback((message, type = 'success') => {
    const id = nextId++
    setItems((prev) => [...prev, { id, message, type, closing: false }])
    window.setTimeout(() => remove(id), VISIBLE_DURATION_MS)
  }, [remove])

  const value = useMemo(() => ({ notify }), [notify])

  return (
    <NotificationContext.Provider value={value}>
      {children}
      <div className="toast-stack" aria-live="polite" aria-atomic="true">
        {items.map((item) => (
          <div
            key={item.id}
            className={`toast toast-${item.type}${item.closing ? ' is-closing' : ''}`}
          >
            <span>{item.message}</span>
            <button
              type="button"
              className="toast-close"
              aria-label={t('dismissNotification')}
              onClick={() => remove(item.id)}
            >
              ×
            </button>
          </div>
        ))}
      </div>
    </NotificationContext.Provider>
  )
}

export const useNotify = () => useContext(NotificationContext)
