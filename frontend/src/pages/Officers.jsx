import { useEffect, useMemo, useState } from 'react'
import api from '../api'
import { useI18n } from '../i18n.jsx'

export default function Officers() {
  const { t } = useI18n()
  const [officers, setOfficers] = useState([])
  const [max, setMax] = useState(1)
  const [query, setQuery] = useState('')

  useEffect(() => {
    api.get('/dashboard').then((r) => {
      setOfficers(r.data.officers)
      setMax(Math.max(1, ...r.data.officers.map((o) => o.final)))
    })
  }, [])

  // Поиск по ФИО; ранг сохраняется по исходному рейтингу.
  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    return officers
      .map((o, i) => ({ ...o, rankNum: i + 1 }))
      .filter((o) => !q || (o.name || '').toLowerCase().includes(q))
  }, [officers, query])

  return (
    <div>
      <div className="page-head">
        <h1>{t('rating')}</h1>
        <div className="page-head-actions">
          <input
            className="search-input"
            type="search"
            placeholder={t('searchName')}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
      </div>
      <div className="card table-card">
        <div className="table-scroll">
        <table className="data-table">
          <thead>
            <tr>
              <th>#</th>
              <th>{t('colOfficer')}</th>
              <th>{t('colRank')}</th>
              <th>{t('colDistrict')}</th>
              <th className="num">{t('confirmedFinal')}</th>
              <th className="num">{t('confirmedCount')}</th>
              <th className="num">{t('pendingShort')}</th>
              <th className="bar-col">{t('byOfficer')}</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((o) => (
              <tr key={o.officer_id}>
                <td className="rank-cell"><span className={`rank-num rank-${o.rankNum}`}>{o.rankNum}</span></td>
                <td className="nowrap strong">{o.name}</td>
                <td className="muted">{o.rank}</td>
                <td className="muted">{o.district}</td>
                <td className="num strong">{o.final}</td>
                <td className="num">{o.confirmed}</td>
                <td className="num">{o.pending}</td>
                <td className="bar-col">
                  <div className="bar-track">
                    <div className="bar-fill" style={{ width: `${(o.final / max) * 100}%` }} />
                  </div>
                </td>
              </tr>
            ))}
            {filtered.length === 0 && (
              <tr><td colSpan={8} className="empty">{officers.length === 0 ? t('noEntries') : t('nothingFound')}</td></tr>
            )}
          </tbody>
        </table>
        </div>
      </div>
    </div>
  )
}
