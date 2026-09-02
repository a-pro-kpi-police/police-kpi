import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import api from '../api'
import { getApiErrorMessage } from '../apiError.js'
import { districtName } from '../districtLocale.js'
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

function initials(name) {
  return (name || '')
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0])
    .join('')
}

export default function AdminSupportChat() {
  const { t, lang } = useI18n()
  const { notify } = useNotify()
  const [conversations, setConversations] = useState([])
  const [activeId, setActiveId] = useState('')
  const [messages, setMessages] = useState([])
  const [query, setQuery] = useState('')
  const [draft, setDraft] = useState('')
  const [loading, setLoading] = useState(true)
  const [messagesLoading, setMessagesLoading] = useState(false)
  const [sending, setSending] = useState(false)
  const activeIdRef = useRef('')
  const messagesRef = useRef(null)
  const textareaRef = useRef(null)

  const loadConversations = useCallback(async ({ silent = false } = {}) => {
    if (!silent) setLoading(true)
    try {
      const { data } = await api.get('/admin/support/conversations')
      const items = Array.isArray(data) ? data : []
      setConversations(items)
      setActiveId((current) => (
        current && items.some((item) => String(item.id) === String(current))
          ? current
          : items[0] ? String(items[0].id) : ''
      ))
    } catch (err) {
      if (!silent) {
        notify(getApiErrorMessage(err, t, 'supportConversationLoadError'), 'error')
        setConversations([])
      }
    } finally {
      if (!silent) setLoading(false)
    }
  }, [notify, t])

  const loadMessages = useCallback(async (conversationId, { silent = false } = {}) => {
    if (!conversationId) {
      setMessages([])
      return
    }
    if (!silent) setMessagesLoading(true)
    try {
      const { data } = await api.get(`/admin/support/conversations/${conversationId}/messages`)
      if (String(activeIdRef.current) !== String(conversationId)) return
      setMessages((current) => mergeSupportMessages(
        current.filter((message) => String(message.conversation_id) === String(conversationId)),
        data.messages,
      ))
    } catch (err) {
      if (!silent) {
        notify(getApiErrorMessage(err, t, 'supportMessagesLoadError'), 'error')
        setMessages([])
      }
    } finally {
      if (!silent) setMessagesLoading(false)
    }
  }, [notify, t])

  useEffect(() => {
    loadConversations()
    const interval = window.setInterval(() => loadConversations({ silent: true }), 15000)
    return () => window.clearInterval(interval)
  }, [loadConversations])

  useEffect(() => {
    activeIdRef.current = activeId
  }, [activeId])

  useEffect(() => {
    if (!activeId) {
      setMessages([])
      return undefined
    }
    setMessages([])
    loadMessages(activeId)
    const interval = window.setInterval(() => loadMessages(activeId, { silent: true }), 5000)
    window.requestAnimationFrame(() => textareaRef.current?.focus())
    return () => window.clearInterval(interval)
  }, [activeId, loadMessages])

  useEffect(() => {
    const node = messagesRef.current
    if (node) node.scrollTop = node.scrollHeight
  }, [messages])

  const filteredConversations = useMemo(() => {
    const needle = query.trim().toLocaleLowerCase()
    if (!needle) return conversations
    return conversations.filter((conversation) => [
      conversation.user?.full_name,
      conversation.user?.username,
      conversation.user?.rank,
      districtName(conversation.user?.district, lang),
      conversation.last_message,
    ].some((value) => (value || '').toLocaleLowerCase().includes(needle)))
  }, [conversations, lang, query])

  const activeConversation = useMemo(
    () => conversations.find((conversation) => String(conversation.id) === String(activeId)),
    [activeId, conversations],
  )

  const send = async (event) => {
    event.preventDefault()
    const body = draft.trim()
    if (!body || !activeId || sending) return
    const submittedConversationId = activeId
    setSending(true)
    try {
      const { data } = await api.post(`/admin/support/conversations/${submittedConversationId}/messages`, { body })
      if (String(activeIdRef.current) === String(submittedConversationId)) {
        setMessages((current) => mergeSupportMessages(
          current.filter((message) => String(message.conversation_id) === String(submittedConversationId)),
          [data],
        ))
        setDraft((current) => (current.trim() === body ? '' : current))
      } else {
        await loadMessages(activeIdRef.current, { silent: true })
      }
      await loadConversations({ silent: true })
    } catch (err) {
      notify(getApiErrorMessage(err, t, 'supportSendError'), 'error')
    } finally {
      setSending(false)
    }
  }

  const timeFormatter = new Intl.DateTimeFormat(
    lang === 'kk' ? 'kk-KZ' : 'ru-RU',
    { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' },
  )

  return (
    <>
      <div className="admin-section-toolbar admin-support-toolbar">
        <input
          className="admin-search-input"
          type="search"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={t('supportSearch')}
          aria-label={t('supportSearch')}
        />
      </div>

      <div className="card admin-support-shell">
        <aside className="admin-support-conversations" aria-label={t('supportConversations')}>
          <div className="admin-support-conversation-list">
            {filteredConversations.map((conversation) => (
              <button
                type="button"
                className={`admin-support-conversation${String(conversation.id) === String(activeId) ? ' active' : ''}`}
                key={conversation.id}
                onClick={() => setActiveId(String(conversation.id))}
              >
                <span className="messenger-contact-avatar" aria-hidden="true">
                  {initials(conversation.user?.full_name)}
                </span>
                <span className="admin-support-conversation-copy">
                  <strong>{conversation.user?.full_name}</strong>
                  <span>
                    {conversation.last_message || conversation.user?.rank || t('supportNoMessagesShort')}
                  </span>
                </span>
                <span className="admin-support-count">
                  {conversation.message_count}
                </span>
              </button>
            ))}
            {!loading && filteredConversations.length === 0 && (
              <div className="admin-support-empty">
                {conversations.length === 0 ? t('supportNoConversations') : t('nothingFound')}
              </div>
            )}
            {loading && <div className="admin-support-empty">{t('loadingEntries')}</div>}
          </div>
        </aside>

        <section className="admin-support-chat">
          {activeConversation ? (
            <>
              <header className="admin-support-chat-head">
                <strong>{activeConversation.user?.full_name}</strong>
                <span>
                  {[activeConversation.user?.rank, districtName(activeConversation.user?.district, lang)]
                    .filter(Boolean)
                    .join(' · ') || activeConversation.user?.username}
                </span>
              </header>
              <div className="admin-support-messages" ref={messagesRef}>
                {messages.map((message) => (
                  <div
                    className={`message-row${message.sent_by_admin ? ' sent' : ' received'}`}
                    key={message.id}
                  >
                    <div className="message-bubble support-message-bubble">
                      {!message.sent_by_admin && (
                        <div className="support-message-author">{message.sender_name}</div>
                      )}
                      {message.sent_by_admin && (
                        <div className="support-message-author">{t('supportAdmin')}</div>
                      )}
                      <p>{message.body}</p>
                      <time dateTime={message.created_at}>
                        {timeFormatter.format(parseApiTimestamp(message.created_at))}
                      </time>
                    </div>
                  </div>
                ))}
                {!messagesLoading && messages.length === 0 && (
                  <div className="messenger-empty">{t('supportNoMessages')}</div>
                )}
                {messagesLoading && <div className="messenger-empty">{t('loadingEntries')}</div>}
              </div>
              <form className="admin-support-compose" onSubmit={send}>
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
                  placeholder={t('supportReplyPlaceholder')}
                  rows={1}
                  maxLength={2000}
                  aria-label={t('supportReplyPlaceholder')}
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
              </form>
            </>
          ) : (
            <div className="messenger-select-empty">
              {t('supportSelectConversation')}
            </div>
          )}
        </section>
      </div>
    </>
  )
}
