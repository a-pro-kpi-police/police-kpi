import { useEffect, useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import api from '../api'
import NameFilterCombobox from '../components/NameFilterCombobox.jsx'
import ProfileHoverCard from '../components/ProfileHoverCard.jsx'
import { districtName } from '../districtLocale.js'
import {
  EFFECTIVENESS_CATEGORIES,
  effectivenessCategory,
  maxEffectivenessScore,
  relativeScorePercent,
} from '../effectiveness.js'
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

function EffectivenessBadge({ category, t }) {
  const labelKey = {
    very_effective: 'categoryVeryEffective',
    effective: 'categoryEffective',
    little_effective: 'categoryLittleEffective',
  }[category]
  return (
    <span className={`effectiveness-badge effectiveness-${category}`}>
      {t(labelKey)}
    </span>
  )
}

export default function Officers() {
  const { t, fmtMonth, lang } = useI18n()
  const [searchParams] = useSearchParams()
  const initialCategory = searchParams.get('category') || ''
  const [officers, setOfficers] = useState([])
  const [months, setMonths] = useState([])
  const [districts, setDistricts] = useState([])
  const [month, setMonth] = useState(searchParams.get('month') || '')
  const [districtFilter, setDistrictFilter] = useState(searchParams.get('district') || '')
  const [officerFilter, setOfficerFilter] = useState('')
  const [categoryFilter, setCategoryFilter] = useState(
    EFFECTIVENESS_CATEGORIES.includes(initialCategory) ? initialCategory : '',
  )
  const [sort, setSort] = useState({ key: 'score', dir: 'desc' })
  const [profileStats, setProfileStats] = useState({})

  useEffect(() => {
    Promise.all([api.get('/months'), api.get('/districts')]).then((responses) => {
      setMonths(responses[0].data)
      setDistricts(responses[1].data)
    })
  }, [])

  useEffect(() => {
    const params = month ? { month } : {}
    Promise.all([
      api.get('/dashboard', { params }),
      api.get('/officers'),
    ]).then(([dashboardResponse, directoryResponse]) => {
      const dashboardById = Object.fromEntries(
        dashboardResponse.data.officers.map((officer) => [officer.officer_id, officer]),
      )
      const directoryById = Object.fromEntries(
        directoryResponse.data.map((officer) => [officer.id, officer]),
      )
      setProfileStats(directoryById)
      const officerRows = directoryResponse.data
        .map((profile) => {
          const dashboard = dashboardById[profile.id] || {}
          const finalScore = Number(dashboard.final) || 0
          return {
            officer_id: profile.id,
            name: profile.full_name,
            rank: profile.rank,
            district: profile.district?.name || '',
            district_id: profile.district?.id || '',
            final: finalScore,
            confirmed: Number(dashboard.confirmed) || 0,
            pending: Number(dashboard.pending) || 0,
            category_score: finalScore,
          }
        })
      setOfficers(officerRows.sort(
        (a, b) => (b.final - a.final) || compareNames(a, b),
      ))
    })
  }, [month])

  const toggleSort = (key) => {
    setSort((current) => (
      current.key === key
        ? { key, dir: current.dir === 'asc' ? 'desc' : 'asc' }
        : { key, dir: 'asc' }
    ))
  }
  const officerOptions = useMemo(
    () => officers
      .filter((officer) => (
        !districtFilter || String(officer.district_id) === String(districtFilter)
      ))
      .map((officer) => ({
      id: officer.officer_id,
      label: officer.name,
      description: `${officer.rank} · ${districtName(officer.district, lang)}`,
    })),
    [districtFilter, lang, officers]
  )

  const categorizedOfficers = useMemo(() => {
    const ratingScope = officers.filter((officer) => (
      !districtFilter || String(officer.district_id) === String(districtFilter)
    ))
    const maxScore = maxEffectivenessScore(ratingScope)
    return officers.map((officer) => ({
      ...officer,
      achievement_percent: relativeScorePercent(officer.final, maxScore),
      effectivenessCategory: effectivenessCategory(officer, maxScore),
    }))
  }, [districtFilter, officers])

  // Номер сохраняется по исходному рейтингу, независимо от выбранной сортировки.
  const filtered = useMemo(() => {
    const rows = categorizedOfficers
      .map((o, i) => ({ ...o, rankNum: i + 1 }))
      .filter((o) => !districtFilter || String(o.district_id) === String(districtFilter))
      .filter((o) => !officerFilter || String(o.officer_id) === String(officerFilter))
      .filter((o) => !categoryFilter || o.effectivenessCategory === categoryFilter)

    const factor = sort.dir === 'asc' ? 1 : -1
    return [...rows].sort((a, b) => {
      const comparison = sort.key === 'score'
        ? a.final - b.final
        : sort.key === 'rank'
          ? rankOrder(a.rank) - rankOrder(b.rank)
          : compareNames(a, b)
      return comparison ? comparison * factor : compareNames(a, b)
    })
  }, [categorizedOfficers, districtFilter, officerFilter, categoryFilter, sort])

  const displayedMonths = month && !months.includes(month)
    ? [month, ...months]
    : months

  return (
    <div>
      <div className="page-head officers-page-head">
        <h1>{t('rating')}</h1>
        <div className="page-head-actions">
          <NameFilterCombobox
            items={officerOptions}
            value={officerFilter}
            onChange={setOfficerFilter}
          />
          <select
            className="select officers-period-select"
            value={month}
            onChange={(event) => setMonth(event.target.value)}
            aria-label={t('period')}
          >
            <option value="">{t('periodAll')}</option>
            {displayedMonths.map((item) => <option key={item} value={item}>{fmtMonth(item)}</option>)}
          </select>
          <select
            className="select officers-district-select"
            value={districtFilter}
            onChange={(event) => {
              setDistrictFilter(event.target.value)
              setOfficerFilter('')
            }}
            aria-label={t('selectGroup')}
          >
            <option value="">{t('allGroups')}</option>
            {districts.map((item) => (
              <option key={item.id} value={item.id}>{districtName(item, lang)}</option>
            ))}
          </select>
          <select
            className="select officers-category-select"
            value={categoryFilter}
            onChange={(event) => setCategoryFilter(event.target.value)}
            aria-label={t('effectivenessCategory')}
          >
            <option value="">{t('allEffectivenessCategories')}</option>
            {EFFECTIVENESS_CATEGORIES.map((item) => (
              <option key={item} value={item}>
                {t({
                  very_effective: 'categoryVeryEffective',
                  effective: 'categoryEffective',
                  little_effective: 'categoryLittleEffective',
                }[item])}
              </option>
            ))}
          </select>
          <select
            className="select officers-score-sort"
            value={sort.key === 'score' ? sort.dir : ''}
            onChange={(event) => setSort({ key: 'score', dir: event.target.value })}
            aria-label={t('sortEffectiveness')}
          >
            {sort.key !== 'score' && <option value="">{t('sortEffectiveness')}</option>}
            <option value="desc">{t('sortBestFirst')}</option>
            <option value="asc">{t('sortWorstFirst')}</option>
          </select>
        </div>
      </div>
      <div className="card table-card">
        <div className="table-scroll officers-table-scroll">
        <table className="data-table officers-table">
          <colgroup>
            <col className="officers-col-position" />
            <col className="officers-col-name" />
            <col className="officers-col-rank" />
            <col className="officers-col-district" />
            <col className="officers-col-score" />
            <col className="officers-col-achievement" />
            <col className="officers-col-category" />
          </colgroup>
          <thead>
            <tr>
              <th>#</th>
              <SortTh column="surname" label={t('colOfficer')} sort={sort} onSort={toggleSort} />
              <SortTh column="rank" label={t('colRank')} sort={sort} onSort={toggleSort} />
              <th>{t('colDistrict')}</th>
              <th className="num">{t('confirmedFinal')}</th>
              <th className="num">{t('colAchievement')}</th>
              <th>{t('effectivenessCategory')}</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((o) => (
              <tr key={o.officer_id}>
                <td className="rank-cell">
                  <span className={`rank-num rank-${o.rankNum}${o.rankNum >= 100 ? ' rank-num-wide' : ''}`}>
                    {o.rankNum}
                  </span>
                </td>
                <td className="strong officer-name-cell">
                  <ProfileHoverCard profile={{
                    id: o.officer_id,
                    name: o.name,
                    rank: o.rank,
                    district: o.district,
                    efficiencyRank: profileStats[o.officer_id]?.efficiency_rank,
                    efficiencyTotal: profileStats[o.officer_id]?.efficiency_total,
                    totalScore: profileStats[o.officer_id]?.efficiency_score,
                    confirmed: o.confirmed,
                  }}>
                    <Link
                      className="officer-dashboard-link"
                      to={`/dashboard/officers/${o.officer_id}`}
                      aria-label={`${t('openOfficerDashboard')}: ${o.name}`}
                      title={t('openOfficerDashboard')}
                    >
                      {o.name}
                    </Link>
                  </ProfileHoverCard>
                </td>
                <td className="muted">{o.rank}</td>
                <td className="muted">{districtName(o.district, lang)}</td>
                <td className="num strong">{o.final}</td>
                <td className="num">
                  {o.achievement_percent}%
                </td>
                <td>
                  <EffectivenessBadge category={o.effectivenessCategory} t={t} />
                </td>
              </tr>
            ))}
            {filtered.length === 0 && (
              <tr><td colSpan={7} className="empty">{officers.length === 0 ? t('noEntries') : t('nothingFound')}</td></tr>
            )}
          </tbody>
        </table>
        </div>
      </div>
    </div>
  )
}
