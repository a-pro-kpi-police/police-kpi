import { useCallback, useEffect, useRef, useState } from 'react'
import api from '../api'
import { getApiErrorMessage } from '../apiError.js'
import { useI18n } from '../i18n.jsx'
import { useNotify } from '../notifications.jsx'
import { mergeSupportMessages, parseApiTimestamp } from '../supportMessages.js'

function SendIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="m3.7 3.9 17.1 7.3a.9.9 0 0 1 0 1.6L3.7 20.1a.9.9 0 0 1-1.2-1l1.2-5.3 9.1-1.8-9.1-1.8-1.2-5.3a.9.9 0 0 1 1.2-1Z" />
    </svg>
  )
}

export default function SupportChat() {
  const { t, lang } = useI18n()
  const { notify } = useNotify()
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
      setMessages((current) => mergeSupportMessages(current, data.messages))
      setError('')
    } catch (err) {
      const message = getApiErrorMessage(err, t, 'supportLoadError')
      if (!silent) setError(message)
    } finally {
      if (!silent) setLoading(false)
    }
  }, [t])

  useEffect(() => {
    loadConversation()
    const interval = window.setInterval(() => loadConversation({ silent: true }), 10000)
    window.requestAnimationFrame(() => textareaRef.current?.focus())
    return () => window.clearInterval(interval)
  }, [loadConversation])

  useEffect(() => {
    const node = messagesRef.current
    if (node) node.scrollTop = node.scrollHeight
  }, [messages])

  const send = async (event) => {
    event.preventDefault()
    const body = draft.trim()
    if (!body || sending) return
    setSending(true)
    setError('')
    try {
      const { data } = await api.post('/support/messages', { body })
      setMessages((current) => mergeSupportMessages(current, [data]))
      setDraft((current) => (current.trim() === body ? '' : current))
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
    <section className="support-page" aria-labelledby="support-title">
      <div className="page-head">
        <h1 id="support-title">{t('supportChat')}</h1>
      </div>
      <div className="card support-chat">
        <header className="support-chat-head">
          <strong>{t('supportAdmin')}</strong>
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
      </div>
    </section>
  )
}
