import { useCallback, useEffect, useRef, useState } from 'react'
import api from '../api'
import { getApiErrorMessage } from '../apiError.js'
import { useI18n } from '../i18n.jsx'
import { useNotify } from '../notifications.jsx'

const parseApiTimestamp = (value) => new Date(
  /(?:Z|[+-]\d{2}:\d{2})$/.test(value) ? value : `${value}Z`,
)

function SupportIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M5 6.5A4.5 4.5 0 0 1 9.5 2h5A4.5 4.5 0 0 1 19 6.5v4A4.5 4.5 0 0 1 14.5 15H13l-4.2 4.2a.7.7 0 0 1-1.2-.5V15A4.5 4.5 0 0 1 5 10.5v-4Z" />
      <path d="M9 7.5h6M9 10.5h4" />
    </svg>
  )
}

function CloseIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="m6 6 12 12M18 6 6 18" />
    </svg>
  )
}

function SendIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="m3.7 3.9 17.1 7.3a.9.9 0 0 1 0 1.6L3.7 20.1a.9.9 0 0 1-1.2-1l1.2-5.3 9.1-1.8-9.1-1.8-1.2-5.3a.9.9 0 0 1 1.2-1Z" />
    </svg>
  )
}

export default function SupportChatWidget() {
  const { t, lang } = useI18n()
  const { notify } = useNotify()
  const [open, setOpen] = useState(false)
  const [messages, setMessages] = useState([])
  const [draft, setDraft] = useState('')
  const [loading, setLoading] = useState(false)
  const [sending, setSending] = useState(false)
  const [error, setError] = useState('')
  const messagesRef = useRef(null)
  const textareaRef = useRef(null)

  const loadConversation = useCallback(async ({ silent = false } = {}) => {
    if (!silent) setLoading(true)
    try {
      const { data } = await api.get('/support/conversation')
      setMessages(Array.isArray(data.messages) ? data.messages : [])
      setError('')
    } catch (err) {
      const message = getApiErrorMessage(err, t, 'supportLoadError')
      if (!silent) setError(message)
    } finally {
      if (!silent) setLoading(false)
    }
  }, [t])

  useEffect(() => {
    if (!open) return undefined
    loadConversation()
    const interval = window.setInterval(() => loadConversation({ silent: true }), 10000)
    window.requestAnimationFrame(() => textareaRef.current?.focus())
    return () => window.clearInterval(interval)
  }, [loadConversation, open])

  useEffect(() => {
    const node = messagesRef.current
    if (node) node.scrollTop = node.scrollHeight
  }, [messages, open])

  const send = async (event) => {
    event.preventDefault()
    const body = draft.trim()
    if (!body || sending) return
    setSending(true)
    setError('')
    try {
      const { data } = await api.post('/support/messages', { body })
      setMessages((current) => [...current, data])
      setDraft('')
    } catch (err) {
      const message = getApiErrorMessage(err, t, 'supportSendError')
      setError(message)
      notify(message, 'error')
    } finally {
      setSending(false)
    }
  }

  const timeFormatter = new Intl.DateTimeFormat(
    lang === 'kk' ? 'kk-KZ' : 'ru-RU',
    { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' },
  )

  return (
    <div className={`support-widget${open ? ' open' : ''}`}>
      <button
        type="button"
        className="support-fab"
        onClick={() => setOpen((current) => !current)}
        aria-label={t(open ? 'supportChatClose' : 'supportChatOpen')}
        title={t(open ? 'supportChatClose' : 'supportChatOpen')}
      >
        {open ? <CloseIcon /> : <SupportIcon />}
      </button>

      {open && (
        <section className="support-window" aria-label={t('supportChat')}>
          <header className="support-window-head">
            <div>
              <strong>{t('supportChat')}</strong>
              <span>{t('supportAdmin')}</span>
            </div>
            <button
              type="button"
              className="support-icon-button"
              onClick={() => setOpen(false)}
              aria-label={t('supportChatClose')}
              title={t('supportChatClose')}
            >
              <CloseIcon />
            </button>
          </header>

          <div className="support-messages" ref={messagesRef}>
            {messages.map((message) => (
              <div
                className={`message-row${message.sent_by_me ? ' sent' : ' received'}`}
                key={message.id}
              >
                <div className="message-bubble support-message-bubble">
                  {!message.sent_by_me && (
                    <div className="support-message-author">
                      {message.sent_by_admin ? t('supportAdmin') : message.sender_name}
                    </div>
                  )}
                  <p>{message.body}</p>
                  <time dateTime={message.created_at}>
                    {timeFormatter.format(parseApiTimestamp(message.created_at))}
                  </time>
                </div>
              </div>
            ))}
            {!loading && messages.length === 0 && (
              <div className="support-empty">{t('supportNoMessages')}</div>
            )}
            {loading && <div className="support-empty">{t('loadingEntries')}</div>}
          </div>

          <form className="support-compose" onSubmit={send}>
            {error && <div className="support-error">{error}</div>}
            <div className="support-compose-row">
              <textarea
                ref={textareaRef}
                value={draft}
                onChange={(event) => setDraft(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' && !event.shiftKey) {
                    event.preventDefault()
                    event.currentTarget.form?.requestSubmit()
                  }
                }}
                placeholder={t('supportMessagePlaceholder')}
                rows={1}
                maxLength={2000}
                aria-label={t('supportMessagePlaceholder')}
              />
              <button
                type="submit"
                className="btn btn-primary message-send-button"
                disabled={!draft.trim() || sending}
                aria-label={t('send')}
                title={t('send')}
              >
                <SendIcon />
              </button>
            </div>
          </form>
        </section>
      )}
    </div>
  )
}
