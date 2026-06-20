import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth.jsx'
import { useI18n } from '../i18n.jsx'

export default function Layout() {
  const { user, logout, isHead } = useAuth()
  const { t, lang, setLang } = useI18n()
  const navigate = useNavigate()

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
          <NavLink to="/" end className="nav-link">{t('dashboard')}</NavLink>
          <NavLink to="/entries" className="nav-link">
            {isHead ? t('entries') : t('myEntries')}
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
              {user.district ? ` · ${user.district.name}` : ''}
            </div>
          </div>
          <div className="topbar-actions">
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
