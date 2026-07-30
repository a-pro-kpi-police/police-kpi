import { useCallback, useEffect, useRef, useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import api from '../api'
import { useAuth } from '../auth.jsx'
import { districtName } from '../districtLocale.js'
import { useI18n } from '../i18n.jsx'
import NotificationCenter from './NotificationCenter.jsx'

function ThemeIcon({ dark }) {
  return dark ? (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2m0 16v2M4.9 4.9l1.4 1.4m11.4 11.4 1.4 1.4M2 12h2m16 0h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
    </svg>
  ) : (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M20 15.3A8.5 8.5 0 0 1 8.7 4a7.7 7.7 0 1 0 11.3 11.3Z" />
    </svg>
  )
}

export default function Layout() {
  const { user, logout, isHead } = useAuth()
  const { t, lang, setLang } = useI18n()
  const navigate = useNavigate()
  const [messageUnread, setMessageUnread] = useState(0)
  const [theme, setTheme] = useState(() => (
    document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light'
  ))
  const themeTransitionTimer = useRef(null)

  const loadMessageUnread = useCallback(async () => {
    try {
      const { data } = await api.get('/messenger/contacts')
      setMessageUnread(data.reduce((sum, contact) => sum + contact.unread_count, 0))
    } catch {
      setMessageUnread(0)
    }
  }, [])

  useEffect(() => {
    loadMessageUnread()
    const interval = window.setInterval(loadMessageUnread, 15000)
    window.addEventListener('messenger-updated', loadMessageUnread)
    return () => {
      window.clearInterval(interval)
      window.removeEventListener('messenger-updated', loadMessageUnread)
    }
  }, [loadMessageUnread])

  useEffect(() => {
    document.documentElement.dataset.theme = theme
    document.documentElement.style.colorScheme = theme
    localStorage.setItem('theme', theme)
  }, [theme])

  useEffect(() => () => {
    if (themeTransitionTimer.current) window.clearTimeout(themeTransitionTimer.current)
    document.documentElement.classList.remove('theme-transitioning')
  }, [])

  const toggleTheme = () => {
    const root = document.documentElement
    if (themeTransitionTimer.current) window.clearTimeout(themeTransitionTimer.current)
    root.classList.remove('theme-transitioning')
    void root.offsetWidth
    root.classList.add('theme-transitioning')
    setTheme((current) => (current === 'dark' ? 'light' : 'dark'))
    themeTransitionTimer.current = window.setTimeout(() => {
      root.classList.remove('theme-transitioning')
      themeTransitionTimer.current = null
    }, 280)
  }

  const doLogout = () => {
    logout()
    navigate('/login')
  }

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="brand">
          <img className="brand-badge" src="/logo.png" alt="" />
          <div>
            <div className="brand-title">{t('appTitle')}</div>
            <div className="brand-sub">{t('appSubtitle')}</div>
          </div>
        </div>
        <nav>
          {isHead && <NavLink to="/dashboard" className="nav-link">{t('dashboard')}</NavLink>}
          <NavLink to="/messenger" className="nav-link nav-link-with-count">
            <span>{t('messenger')}</span>
            {messageUnread > 0 && <span className="nav-unread-count">{messageUnread}</span>}
          </NavLink>
          <NavLink to="/entries" className="nav-link">
            {isHead ? t('entries') : t('myEntries')}
          </NavLink>
          <NavLink to="/targets" className="nav-link">
            {isHead ? t('targets') : t('myTargets')}
          </NavLink>
          {isHead && <NavLink to="/officers" className="nav-link">{t('officers')}</NavLink>}
        </nav>
      </aside>
      <main className="content">
        <header className="topbar">
          <div className="user-box">
            <div className="user-name">{user.full_name}</div>
            <div className="user-meta">
              {user.rank} · {t(`role_${user.role}`)}
              {user.district ? ` · ${districtName(user.district, lang)}` : ''}
            </div>
          </div>
          <div className="topbar-actions">
            <NotificationCenter />
            <button
              type="button"
              className="theme-toggle"
              onClick={toggleTheme}
              aria-pressed={theme === 'dark'}
              aria-label={t(theme === 'dark' ? 'switchToLightMode' : 'switchToDarkMode')}
              title={t(theme === 'dark' ? 'switchToLightMode' : 'switchToDarkMode')}
            >
              <ThemeIcon dark={theme === 'dark'} />
            </button>
            <div className="lang-switch">
              <button className={lang === 'ru' ? 'active' : ''} onClick={() => setLang('ru')}>RU</button>
              <button className={lang === 'kk' ? 'active' : ''} onClick={() => setLang('kk')}>KZ</button>
            </div>
            <button className="btn btn-ghost" onClick={doLogout}>{t('logout')}</button>
          </div>
        </header>
        <div className="page">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
