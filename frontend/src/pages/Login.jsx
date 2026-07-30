import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../auth.jsx'
import { useI18n } from '../i18n.jsx'
import { useNotify } from '../notifications.jsx'
import LanguageSwitch from '../components/LanguageSwitch.jsx'

export default function Login() {
  const { login, user } = useAuth()
  const { t } = useI18n()
  const { notify } = useNotify()
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    if (user) navigate('/', { replace: true })
  }, [navigate, user])

  const submit = async (e) => {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      const loggedInUser = await login(username, password)
      notify(`${t('welcome')}, ${loggedInUser.full_name}!`, 'welcome')
      navigate('/', { replace: true })
    } catch {
      setError(t('loginError'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="login-page">
      <div className="login-card">
        <LanguageSwitch className="login-lang" />
        <img className="login-badge" src="/logo.png" alt="" />
        <h1>{t('appTitle')}</h1>
        <p className="login-sub">{t('appSubtitle')}</p>
        <form onSubmit={submit}>
          <label>{t('username')}</label>
          <input value={username} onChange={(e) => setUsername(e.target.value)} autoFocus />
          <label>{t('password')}</label>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
          {error && <div className="error">{error}</div>}
          <button className="btn btn-primary btn-block" disabled={busy}>{t('signIn')}</button>
        </form>
      </div>
    </div>
  )
}
