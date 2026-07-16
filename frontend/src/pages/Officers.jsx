import { useEffect, useMemo, useState } from 'react'
import api from '../api'
import { useI18n } from '../i18n.jsx'

const RANK_ORDER = [
  'Сержант полиции',
  'Старший сержант полиции',
  'Лейтенант полиции',
  'Старший лейтенант полиции',
  'Капитан полиции',
  'Майор полиции',
  'Подполковник полиции',
  'Полковник полиции',
]

const rankOrder = (rank) => {
  const index = RANK_ORDER.indexOf(rank)
  return index === -1 ? RANK_ORDER.length : index
}

const compareNames = (a, b) => (a.name || '').localeCompare(b.name || '', 'ru')

function SortTh({ column, label, sort, onSort }) {
  const active = sort.key === column
  return (
    <th
      className={`sortable${active ? ' sorted' : ''}`}
      aria-sort={active ? (sort.dir === 'asc' ? 'ascending' : 'descending') : 'none'}
      onClick={() => onSort(column)}
    >
      {label}
      <span className="sort-ind">{active ? (sort.dir === 'asc' ? '▲' : '▼') : '⇅'}</span>
    </th>
  )
}

export default function Officers() {
  const { t } = useI18n()
  const [officers, setOfficers] = useState([])
  const [max, setMax] = useState(1)
  const [query, setQuery] = useState('')
  const [sort, setSort] = useState({ key: null, dir: 'asc' })

  useEffect(() => {
    api.get('/dashboard').then((r) => {
      setOfficers(r.data.officers)
      setMax(Math.max(1, ...r.data.officers.map((o) => o.final)))
    })
  }, [])

  const toggleSort = (key) => {
    setSort((current) => (
      current.key === key
        ? { key, dir: current.dir === 'asc' ? 'desc' : 'asc' }
        : { key, dir: 'asc' }
    ))
  }

  // Номер сохраняется по исходному рейтингу, независимо от выбранной сортировки.
  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    const rows = officers
      .map((o, i) => ({ ...o, rankNum: i + 1 }))
      .filter((o) => !q || (o.name || '').toLowerCase().includes(q))

    if (!sort.key) return rows
    const factor = sort.dir === 'asc' ? 1 : -1
    return [...rows].sort((a, b) => {
      const comparison = sort.key === 'rank'
        ? rankOrder(a.rank) - rankOrder(b.rank)
        : compareNames(a, b)
      return (comparison || compareNames(a, b)) * factor
    })
  }, [officers, query, sort])

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
              <SortTh column="surname" label={t('colOfficer')} sort={sort} onSort={toggleSort} />
              <SortTh column="rank" label={t('colRank')} sort={sort} onSort={toggleSort} />
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
