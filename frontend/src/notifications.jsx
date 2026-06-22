import { createContext, useContext, useMemo, useState } from 'react'

const NotificationContext = createContext(null)

let nextId = 1

export function NotificationProvider({ children }) {
  const [items, setItems] = useState([])

  const remove = (id) => {
    setItems((prev) => prev.filter((item) => item.id !== id))
  }

  const notify = (message, type = 'success') => {
    const id = nextId++
    setItems((prev) => [...prev, { id, message, type }])
    window.setTimeout(() => remove(id), 3500)
  }

  const value = useMemo(() => ({ notify }), [])

  return (
    <NotificationContext.Provider value={value}>
      {children}
      <div className="toast-stack" aria-live="polite" aria-atomic="true">
        {items.map((item) => (
          <div key={item.id} className={`toast toast-${item.type}`}>
            <span>{item.message}</span>
            <button type="button" className="toast-close" onClick={() => remove(item.id)}>
              ×
            </button>
          </div>
        ))}
      </div>
    </NotificationContext.Provider>
  )
}

export const useNotify = () => useContext(NotificationContext)
