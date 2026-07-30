import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { districtName } from '../districtLocale.js'
import { useI18n } from '../i18n.jsx'

const initialsFor = (name = '') => (
  name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0])
    .join('')
    .toUpperCase()
)

export default function ProfileHoverCard({ profile, children }) {
  const { t, lang } = useI18n()
  const triggerRef = useRef(null)
  const closeTimerRef = useRef(null)
  const [open, setOpen] = useState(false)
  const [position, setPosition] = useState({ left: 0, top: 0 })

  const cancelClose = () => {
    if (closeTimerRef.current) window.clearTimeout(closeTimerRef.current)
  }
  const show = () => {
    cancelClose()
    setOpen(true)
  }
  const scheduleClose = () => {
    cancelClose()
    closeTimerRef.current = window.setTimeout(() => setOpen(false), 90)
  }

  useEffect(() => {
    if (!open) return undefined
    const updatePosition = () => {
      const trigger = triggerRef.current
      if (!trigger) return
      const rect = trigger.getBoundingClientRect()
      const width = 286
      const estimatedHeight = 178
      const left = Math.min(
        Math.max(12, rect.left),
        Math.max(12, window.innerWidth - width - 12),
      )
      const below = rect.bottom + 8
      const top = below + estimatedHeight <= window.innerHeight
        ? below
        : Math.max(12, rect.top - estimatedHeight - 8)
      setPosition({ left, top })
    }
    updatePosition()
    window.addEventListener('resize', updatePosition)
    window.addEventListener('scroll', updatePosition, true)
    return () => {
      window.removeEventListener('resize', updatePosition)
      window.removeEventListener('scroll', updatePosition, true)
    }
  }, [open])

  useEffect(() => () => cancelClose(), [])

  return (
    <>
      <span
        ref={triggerRef}
        className="profile-hover-trigger"
        onMouseEnter={show}
        onMouseLeave={scheduleClose}
        onFocus={show}
        onBlur={scheduleClose}
      >
        {children}
      </span>
      {open && createPortal(
        <aside
          className="mini-profile"
          style={position}
          role="tooltip"
          onMouseEnter={cancelClose}
          onMouseLeave={scheduleClose}
        >
          <div className="mini-profile-main">
            <div className="mini-profile-avatar" aria-hidden="true">
              {initialsFor(profile?.name)}
            </div>
            <div>
              <div className="mini-profile-name">{profile?.name || '—'}</div>
              <div className="mini-profile-rank">{profile?.rank || t('role_officer')}</div>
            </div>
          </div>
          <dl className="mini-profile-details">
            <div>
              <dt>{t('colDistrict')}</dt>
              <dd>{districtName(profile?.district, lang) || '—'}</dd>
            </div>
            {profile?.efficiencyRank !== undefined && profile?.efficiencyRank !== null && (
              <div>
                <dt>{t('effectivenessPlace')}</dt>
                <dd>#{profile.efficiencyRank} / {profile.efficiencyTotal || '—'}</dd>
              </div>
            )}
            {profile?.totalScore !== undefined && profile?.totalScore !== null ? (
              <div>
                <dt>{t('overallPoints')}</dt>
                <dd>{profile.totalScore}</dd>
              </div>
            ) : profile?.score !== undefined && (
              <div>
                <dt>{t('confirmedFinal')}</dt>
                <dd>{profile.score}</dd>
              </div>
            )}
            {profile?.confirmed !== undefined && (
              <div>
                <dt>{t('confirmedCount')}</dt>
                <dd>{profile.confirmed}</dd>
              </div>
            )}
          </dl>
        </aside>,
        document.body,
      )}
    </>
  )
}
