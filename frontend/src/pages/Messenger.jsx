import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import api from '../api'
import { useAuth } from '../auth.jsx'
import NameFilterCombobox from '../components/NameFilterCombobox.jsx'
import ProfileHoverCard from '../components/ProfileHoverCard.jsx'
import { useI18n } from '../i18n.jsx'
import { kpiIndicator } from '../kpiLocale.js'

const parseApiTimestamp = (value) => new Date(
  /(?:Z|[+-]\d{2}:\d{2})$/.test(value) ? value : `${value}Z`,
)

function SendIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="m3.7 3.9 17.1 7.3a.9.9 0 0 1 0 1.6L3.7 20.1a.9.9 0 0 1-1.2-1l1.2-5.3 9.1-1.8-9.1-1.8-1.2-5.3a.9.9 0 0 1 1.2-1Z" />
    </svg>
  )
}

function TrashIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M8 9v8m4-8v8m4-8v8M5 6h14m-9-3h4l1 3H9l1-3Zm-3 3 1 14h8l1-14" />
    </svg>
  )
}

function DeletedIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="m5 5 14 14M7.5 7.5A6.4 6.4 0 0 0 5 12c1.7 3 4 4.5 7 4.5 1.1 0 2.1-.2 3-.7M10 5.7c.6-.1 1.3-.2 2-.2 3 0 5.3 1.5 7 4.5-.5.9-1.1 1.7-1.8 2.3" />
    </svg>
  )
}

function ReferenceSummary({ entry, dateFormatter, lang, t }) {
  return (
    <>
      <span>{t('kpiRecord')} #{entry.id}</span>
      <strong>{entry.code} · {kpiIndicator(entry, lang)}</strong>
      <small>
        {dateFormatter.format(new Date(`${entry.date}T00:00:00`))}
        {' · '}{entry.final_score} {t('points')}
        {' · '}{t(entry.status)}
      </small>
    </>
  )
}

function KpiMessageReference({ entry, dateFormatter, expandable = false, lang, t }) {
  const [expanded, setExpanded] = useState(false)
  if (!entry) return null
  return (
    <div className={`message-kpi-reference${expanded ? ' expanded' : ''}`}>
      {expandable ? (
        <button
          type="button"
          className="message-kpi-reference-summary"
          onClick={() => setExpanded((current) => !current)}
          aria-expanded={expanded}
          aria-label={t(expanded ? 'collapseKpiDetails' : 'expandKpiDetails')}
          title={t(expanded ? 'collapseKpiDetails' : 'expandKpiDetails')}
        >
          <span className="message-kpi-reference-copy">
            <ReferenceSummary entry={entry} dateFormatter={dateFormatter} lang={lang} t={t} />
          </span>
          <span className="message-kpi-reference-chevron" aria-hidden="true">
            {expanded ? '⌃' : '⌄'}
          </span>
        </button>
      ) : (
        <div className="message-kpi-reference-copy">
          <ReferenceSummary entry={entry} dateFormatter={dateFormatter} lang={lang} t={t} />
        </div>
      )}
      {expandable && expanded && (
        <div className="message-kpi-details">
          <dl>
            <div className="wide">
              <dt>{t('colBlock')}</dt>
              <dd>{entry.block || '—'}</dd>
            </div>
            <div className="wide">
              <dt>{t('colNpa')}</dt>
              <dd>{entry.npa || '—'}</dd>
            </div>
            <div>
              <dt>{t('colQuantity')}</dt>
              <dd>{entry.quantity}</dd>
            </div>
            <div>
              <dt>{t('colPointsPerUnit')}</dt>
              <dd>{entry.points_per_unit}</dd>
            </div>
            <div>
              <dt>{t('colCoeff')}</dt>
              <dd>{entry.coeff}</dd>
            </div>
            <div>
              <dt>{t('colRaw')}</dt>
              <dd>{entry.raw_score}</dd>
            </div>
            <div className="wide message-kpi-detail-total">
              <dt>{t('colFinal')}</dt>
              <dd>{entry.final_score}</dd>
            </div>
            <div className="wide">
              <dt>{t('colSource')}</dt>
              <dd>
                {entry.source_url ? (
                  <a href={entry.source_url} target="_blank" rel="noreferrer">
                    {t('openSource')}
                  </a>
                ) : '—'}
              </dd>
            </div>
          </dl>
          {entry.comments && (
            <div className="message-kpi-detail-comment">
              <span>{t('colComments')}</span>
              <p>{entry.comments}</p>
            </div>
          )}
          {entry.head_comment && (
            <div className="message-kpi-detail-comment">
              <span>{t('headResolution')}</span>
              <p>{entry.head_comment}</p>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export default function Messenger() {
  const { user, isHead } = useAuth()
  const { t, lang } = useI18n()
  const [searchParams] = useSearchParams()
  const requestedContactId = searchParams.get('contact') || ''
  const requestedEntryId = searchParams.get('entry') || ''
  const [contacts, setContacts] = useState([])
  const [activeId, setActiveId] = useState('')
  const [messages, setMessages] = useState([])
  const [discussionEntries, setDiscussionEntries] = useState([])
  const [selectedEntryId, setSelectedEntryId] = useState('')
  const [draft, setDraft] = useState('')
  const [sending, setSending] = useState(false)
  const messagesRef = useRef(null)
  const textareaRef = useRef(null)
  const contactRefs = useRef(new Map())
  const requestedContactAppliedRef = useRef(false)
  const requestedEntryAppliedRef = useRef(false)

  const contactOptions = useMemo(
    () => contacts.map((contact) => ({
      id: contact.id,
      label: contact.full_name,
      description: contact.rank,
    })),
    [contacts],
  )
  const activeContact = useMemo(
    () => contacts.find((contact) => String(contact.id) === String(activeId)),
    [activeId, contacts],
  )
  const selectedEntry = useMemo(
    () => discussionEntries.find((entry) => String(entry.id) === String(selectedEntryId)),
    [discussionEntries, selectedEntryId],
  )

  const loadContacts = useCallback(async () => {
    const { data } = await api.get('/messenger/contacts')
    setContacts(data)
    setActiveId((current) => {
      if (
        !requestedContactAppliedRef.current
        && requestedContactId
        && data.some((contact) => String(contact.id) === String(requestedContactId))
      ) {
        requestedContactAppliedRef.current = true
        return String(requestedContactId)
      }
      return current && data.some((contact) => String(contact.id) === String(current))
        ? current
        : data[0] ? String(data[0].id) : ''
    })
  }, [requestedContactId])

  const loadMessages = useCallback(async (contactId) => {
    if (!contactId) {
      setMessages([])
      return
    }
    const { data } = await api.get(`/messenger/messages/${contactId}`)
    setMessages(data)
    window.dispatchEvent(new CustomEvent('messenger-updated'))
  }, [])

  const loadDiscussionEntries = useCallback(async (contactId) => {
    if (!contactId) {
      setDiscussionEntries([])
      return
    }
    const { data } = await api.get(`/messenger/entries/${contactId}`)
    setDiscussionEntries(data)
    return data
  }, [])

  useEffect(() => {
    loadContacts()
    const interval = window.setInterval(loadContacts, 10000)
    return () => window.clearInterval(interval)
  }, [loadContacts])

  useEffect(() => {
    if (!activeId) return undefined
    let cancelled = false
    setSelectedEntryId('')
    loadDiscussionEntries(activeId).then((entries) => {
      if (
        !cancelled
        && !requestedEntryAppliedRef.current
        && requestedEntryId
        && (!requestedContactId || String(activeId) === String(requestedContactId))
        && entries.some((entry) => String(entry.id) === String(requestedEntryId))
      ) {
        requestedEntryAppliedRef.current = true
        setSelectedEntryId(String(requestedEntryId))
        window.requestAnimationFrame(() => textareaRef.current?.focus())
      }
    })
    loadMessages(activeId)
    const interval = window.setInterval(() => loadMessages(activeId), 5000)
    return () => {
      cancelled = true
      window.clearInterval(interval)
    }
  }, [
    activeId,
    loadDiscussionEntries,
    loadMessages,
    requestedContactId,
    requestedEntryId,
  ])

  useEffect(() => {
    const node = messagesRef.current
    if (node) node.scrollTop = node.scrollHeight
  }, [messages])

  useEffect(() => {
    contactRefs.current.get(String(activeId))?.scrollIntoView({
      block: 'nearest',
    })
  }, [activeId])

  const send = async (event) => {
    event.preventDefault()
    const body = draft.trim()
    if (!body || !activeId || sending) return
    setSending(true)
    try {
      const { data } = await api.post('/messenger/messages', {
        recipient_id: Number(activeId),
        body,
        entry_id: selectedEntryId ? Number(selectedEntryId) : null,
      })
      setMessages((current) => [...current, data])
      setDraft('')
      setSelectedEntryId('')
      await loadContacts()
      window.dispatchEvent(new CustomEvent('messenger-updated'))
    } finally {
      setSending(false)
    }
  }

  const removeMessage = async (message) => {
    if (!message.sent_by_me || message.is_deleted) return
    if (!window.confirm(t('confirmDeleteMessage'))) return
    const { data } = await api.delete(`/messenger/messages/${message.id}`)
    setMessages((current) => current.map((item) => (
      item.id === data.id ? data : item
    )))
    await loadContacts()
    window.dispatchEvent(new CustomEvent('messenger-updated'))
  }

  const timeFormatter = new Intl.DateTimeFormat(
    lang === 'kk' ? 'kk-KZ' : 'ru-RU',
    { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' },
  )
  const entryDateFormatter = new Intl.DateTimeFormat(
    lang === 'kk' ? 'kk-KZ' : 'ru-RU',
    { day: '2-digit', month: '2-digit', year: 'numeric' },
  )

  return (
    <div className="messenger-page">
      <div className="page-head">
        <h1>{t('messenger')}</h1>
      </div>
      <div className="card messenger-shell">
        <aside className="messenger-contacts" aria-label={t('conversations')}>
          <div className="messenger-contacts-head">
            <span>{isHead ? t('employees') : t('headContact')}</span>
            <NameFilterCombobox
              items={contactOptions}
              value={activeId}
              onChange={(contactId) => {
                if (contactId) setActiveId(contactId)
              }}
              includeAll={false}
              placeholder={t('searchName')}
              ariaLabel={t('searchName')}
            />
          </div>
          <div className="messenger-contact-list">
            {contacts.map((contact) => (
              <button
                type="button"
                className={`messenger-contact${String(contact.id) === String(activeId) ? ' active' : ''}`}
                key={contact.id}
                ref={(node) => {
                  if (node) {
                    contactRefs.current.set(String(contact.id), node)
                  } else {
                    contactRefs.current.delete(String(contact.id))
                  }
                }}
                onClick={() => setActiveId(String(contact.id))}
              >
                <span className="messenger-contact-avatar" aria-hidden="true">
                  {contact.full_name.split(/\s+/).slice(0, 2).map((part) => part[0]).join('')}
                </span>
                <span className="messenger-contact-copy">
                  <ProfileHoverCard profile={{
                    id: contact.id,
                    name: contact.full_name,
                    rank: contact.rank,
                    district: contact.district_name,
                    efficiencyRank: contact.efficiency_rank,
                    efficiencyTotal: contact.efficiency_total,
                    totalScore: contact.efficiency_score,
                  }}>
                    <strong>{contact.full_name}</strong>
                  </ProfileHoverCard>
                  <span>
                    {contact.last_message_deleted
                      ? t('messageDeleted')
                      : contact.last_message || contact.rank}
                  </span>
                </span>
                {contact.unread_count > 0 && (
                  <span className="messenger-unread">{contact.unread_count}</span>
                )}
              </button>
            ))}
            {!contacts.length && <div className="messenger-contact-empty">{t('noContacts')}</div>}
          </div>
        </aside>

        <section className="messenger-conversation">
          {activeContact ? (
            <>
              <header className="messenger-conversation-head">
                <ProfileHoverCard profile={{
                  id: activeContact.id,
                  name: activeContact.full_name,
                  rank: activeContact.rank,
                  district: activeContact.district_name,
                  efficiencyRank: activeContact.efficiency_rank,
                  efficiencyTotal: activeContact.efficiency_total,
                  totalScore: activeContact.efficiency_score,
                }}>
                  <strong>{activeContact.full_name}</strong>
                </ProfileHoverCard>
                <span>{activeContact.rank}</span>
              </header>
              <div className="messenger-messages" ref={messagesRef}>
                {messages.map((message) => (
                  <div
                    className={`message-row${message.sent_by_me ? ' sent' : ' received'}`}
                    key={message.id}
                  >
                    <div className={`message-bubble${message.is_deleted ? ' deleted' : ''}`}>
                      {message.sent_by_me && !message.is_deleted && (
                        <button
                          type="button"
                          className="message-delete-button"
                          onClick={() => removeMessage(message)}
                          aria-label={t('deleteMessage')}
                          title={t('deleteMessage')}
                        >
                          <TrashIcon />
                        </button>
                      )}
                      {message.is_deleted ? (
                        <p className="message-deleted-copy">
                          <DeletedIcon />
                          <span>{t('messageDeleted')}</span>
                        </p>
                      ) : (
                        <>
                          <KpiMessageReference
                            entry={message.entry}
                            dateFormatter={entryDateFormatter}
                            expandable
                            lang={lang}
                            t={t}
                          />
                          <p>{message.body}</p>
                        </>
                      )}
                      <time dateTime={message.created_at}>
                        {timeFormatter.format(parseApiTimestamp(message.created_at))}
                      </time>
                    </div>
                  </div>
                ))}
                {!messages.length && (
                  <div className="messenger-empty">{t('noMessages')}</div>
                )}
              </div>
              <form className="messenger-compose" onSubmit={send}>
                <div className="messenger-compose-reference-picker">
                  <label htmlFor="messenger-entry-reference">{t('discussKpiRecord')}</label>
                  <select
                    id="messenger-entry-reference"
                    value={selectedEntryId}
                    onChange={(event) => setSelectedEntryId(event.target.value)}
                    aria-label={t('discussKpiRecord')}
                  >
                    <option value="">{t('noKpiReference')}</option>
                    {discussionEntries.map((entry) => (
                      <option key={entry.id} value={entry.id}>
                        #{entry.id} · {entry.code} · {entryDateFormatter.format(new Date(`${entry.date}T00:00:00`))}
                        {' · '}{entry.final_score} {t('points')}
                      </option>
                    ))}
                  </select>
                </div>
                {selectedEntry && (
                  <div className="messenger-compose-reference-preview">
                    <KpiMessageReference
                      entry={selectedEntry}
                      dateFormatter={entryDateFormatter}
                      lang={lang}
                      t={t}
                    />
                    <button
                      type="button"
                      onClick={() => setSelectedEntryId('')}
                      aria-label={t('clearKpiReference')}
                      title={t('clearKpiReference')}
                    >
                      ×
                    </button>
                  </div>
                )}
                <div className="messenger-compose-row">
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
                    placeholder={t('messagePlaceholder')}
                    rows={1}
                    maxLength={2000}
                    aria-label={t('messagePlaceholder')}
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
            </>
          ) : (
            <div className="messenger-select-empty">
              {t('selectConversation')}
            </div>
          )}
        </section>
      </div>
    </div>
  )
}
