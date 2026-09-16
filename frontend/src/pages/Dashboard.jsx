import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  Bar, BarChart, CartesianGrid, Cell, LabelList, Legend, Line, LineChart,
  ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis,
} from 'recharts'
import api from '../api'
import { getApiErrorMessage } from '../apiError.js'
import { adminOfficerDashboardEndpoints } from '../adminOfficerDashboard.js'
import { useAuth } from '../auth.jsx'
import {
  blockScoreDomain,
  blockScoreLabelPosition,
  blockScoreRows,
  formatBlockScore,
} from '../blockScoreChart.js'
import NameFilterCombobox from '../components/NameFilterCombobox.jsx'
import ProfileHoverCard from '../components/ProfileHoverCard.jsx'
import { districtName } from '../districtLocale.js'
import {
  compareEffectiveness,
  EFFECTIVENESS_CATEGORIES,
  effectivenessCategory,
  averageEffectivenessScore,
  relativeScorePercent,
} from '../effectiveness.js'
import { useI18n } from '../i18n.jsx'
import { kpiIndicator } from '../kpiLocale.js'

const COLORS = ['#1e4d8c', '#2f7d4f', '#b8860b', '#9c3848', '#5b4b8a', '#0f8b8d', '#a85b2b', '#3b6ea5']
const BLOCK_COLORS = ['#1e4d8c', '#2f7d4f', '#b8860b', '#5b4b8a', '#0f8b8d', '#a85b2b', '#3b6ea5']
const STATUS_COLORS = { confirmed: '#2f7d4f', pending: '#b8860b', rejected: '#9c3848' }
const EFFECTIVENESS_COLORS = {
  very_effective: '#2f7d4f',
  effective: '#b8860b',
  little_effective: '#9c3848',
}
const CHART_CURSOR = { fill: 'var(--chart-hover)' }
const DONUT_RADIUS = 40
const DONUT_STROKE_WIDTH = 17

function StatCard({ label, value, accent }) {
  return (
    <div className="stat-card" style={{ borderTopColor: accent }}>
      <div className="stat-value">{value}</div>
      <div className="stat-label">{label}</div>
    </div>
  )
}

function BlockAxisTick({ y, payload, maxLength }) {
  const value = String(payload.value)
  const displayValue = maxLength && value.length > maxLength
    ? `${value.slice(0, maxLength - 1)}…`
    : value

  return (
    <text
      className="block-axis-label"
      x={6}
      y={y}
      dy="0.35em"
      fill="#6b7785"
      fontSize={11}
      textAnchor="start"
    >
      <title>{value}</title>
      {displayValue}
    </text>
  )
}

function useCompactChartLayout() {
  const [compact, setCompact] = useState(
    () => typeof window !== 'undefined' && window.matchMedia('(max-width: 620px)').matches,
  )

  useEffect(() => {
    const media = window.matchMedia('(max-width: 620px)')
    const update = () => setCompact(media.matches)
    update()
    media.addEventListener('change', update)
    return () => media.removeEventListener('change', update)
  }, [])

  return compact
}

function BlockScoreLabel({ x, y, width, height, value, pointsLabel }) {
  const label = formatBlockScore(value, pointsLabel)
  const position = blockScoreLabelPosition({ x, width, value })

  return (
    <text
      className="block-score-label"
      x={position.x}
      y={y + height / 2}
      dy="0.35em"
      textAnchor={position.textAnchor}
    >
      {label}
    </text>
  )
}

function IndicatorVolumeTooltip({ active, payload, t, lang }) {
  if (!active || !payload?.length) return null
  const item = payload[0].payload
  return (
    <div className="indicator-volume-tooltip">
      <div className="indicator-volume-tooltip-title">{kpiIndicator(item, lang) || item.code}</div>
      <div className="indicator-volume-tooltip-value">
        {t('unitsCompleted')}: <strong>{item.quantity}</strong>
      </div>
    </div>
  )
}

function StatusDonut({
  items,
  total,
  label,
  order = ['confirmed', 'rejected', 'pending'],
}) {
  const baseSegments = order
    .map((key) => items.find((item) => item.key === key))
    .filter(Boolean)
    .map((item) => ({
      ...item,
      rawPercent: (item.value / total) * 100,
    }))
  const dominantIndex = baseSegments.reduce(
    (bestIndex, segment, index, all) => (
      segment.value > all[bestIndex].value ? index : bestIndex
    ),
    0,
  )
  const smallestValue = Math.min(...baseSegments.map((segment) => segment.value))
  const displayPercents = baseSegments.map((segment, index) => {
    if (index === dominantIndex || segment.rawPercent >= 1) {
      return segment.rawPercent
    }
    return Math.max(
      segment.rawPercent,
      Math.min(4, 0.75 * (segment.value / smallestValue)),
    )
  })
  if (baseSegments.length > 1) {
    displayPercents[dominantIndex] = Math.max(
      0,
      100 - displayPercents.reduce(
        (sum, percent, index) => (
          index === dominantIndex ? sum : sum + percent
        ),
        0,
      ),
    )
  }

  let offset = 0
  const segments = baseSegments
    .map((item, index) => {
      const percent = displayPercents[index]
      const segment = { ...item, offset, percent }
      offset += percent
      return segment
    })

  if (segments.length === 1) {
    return (
      <svg className="overview-status-donut" viewBox="0 0 120 120" role="img" aria-label={label}>
        <circle
          className="overview-status-segment"
          cx="60"
          cy="60"
          r={DONUT_RADIUS}
          pathLength="100"
          fill="none"
          stroke={segments[0].color}
          strokeWidth={DONUT_STROKE_WIDTH}
          strokeDasharray="100 0"
        >
          <title>{segments[0].label}: {segments[0].value}</title>
        </circle>
      </svg>
    )
  }

  return (
    <svg className="overview-status-donut" viewBox="0 0 120 120" role="img" aria-label={label}>
      {segments.map((segment) => (
        <circle
          key={segment.key}
          className="overview-status-segment"
          cx="60"
          cy="60"
          r={DONUT_RADIUS}
          pathLength="100"
          fill="none"
          stroke={segment.color}
          strokeWidth={DONUT_STROKE_WIDTH}
          strokeDasharray={`${segment.percent} ${100 - segment.percent}`}
          strokeDashoffset={-segment.offset}
          strokeLinecap="butt"
          transform="rotate(180 60 60)"
        >
          <title>{segment.label}: {segment.value}</title>
        </circle>
      ))}
    </svg>
  )
}

function EffectivenessOverviewChart({ officers, t }) {
  const averageScore = averageEffectivenessScore(officers)
  const counts = Object.fromEntries(
    EFFECTIVENESS_CATEGORIES.map((category) => [category, 0]),
  )
  for (const officer of officers) {
    counts[effectivenessCategory(officer, averageScore)] += 1
  }
  const labels = {
    very_effective: t('categoryVeryEffective'),
    effective: t('categoryEffective'),
    little_effective: t('categoryLittleEffective'),
  }
  const items = EFFECTIVENESS_CATEGORIES.map((category) => ({
    key: category,
    label: labels[category],
    value: counts[category],
    color: EFFECTIVENESS_COLORS[category],
  }))
  const chartItems = items.filter((item) => item.value > 0)
  const total = officers.length

  return (
    <section className="card overview-chart-card overview-status-card effectiveness-overview-card">
      <h3>{t('effectivenessDistribution')}</h3>
      <div className="overview-status-body">
        <div className="overview-status-chart">
          {total > 0 ? (
            <StatusDonut
              items={chartItems}
              total={total}
              label={`${t('effectivenessDistribution')}: ${total}`}
              order={EFFECTIVENESS_CATEGORIES}
            />
          ) : (
            <div className="overview-empty-chart">{t('nothingFound')}</div>
          )}
        </div>
        <div className="overview-status-legend">
          {items.map((item) => (
            <div key={item.key}>
              <span
                className="overview-legend-swatch"
                style={{ background: item.color }}
                aria-hidden="true"
              />
              <span>{item.label}</span>
              <strong>{item.value}</strong>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

function OverviewCharts({ data, t }) {
  const statusData = [
    { key: 'confirmed', label: t('confirmedCount'), value: data.confirmed_count, color: STATUS_COLORS.confirmed },
    { key: 'pending', label: t('pendingCount'), value: data.pending_count, color: STATUS_COLORS.pending },
    { key: 'rejected', label: t('rejectedCount'), value: data.rejected_count, color: STATUS_COLORS.rejected },
  ]
  const totalRecords = statusData.reduce((sum, item) => sum + item.value, 0)
  const confirmedRateRecords = data.confirm_rate_confirmed_count ?? data.confirmed_count
  const decidedRecords = data.confirm_rate_decided_count ?? (data.confirmed_count + data.rejected_count)
  const statusChartData = statusData.filter((item) => item.value > 0)
  const confirmationRate = Math.min(100, Math.max(0, Number(data.confirm_rate) || 0))

  return (
    <div className="overview-visual-grid">
      <section className="card overview-chart-card overview-status-card">
        <h3>{t('entryStatusOverview')}</h3>
        <div className="overview-status-body">
          <div className="overview-status-chart">
            {totalRecords > 0 ? (
              <StatusDonut
                items={statusChartData}
                total={totalRecords}
                label={`${t('entryStatusOverview')}: ${totalRecords}`}
              />
            ) : (
              <div className="overview-empty-chart">{t('nothingFound')}</div>
            )}
          </div>
          <div className="overview-status-legend">
            {statusData.map((item) => (
              <div key={item.key}>
                <span className="overview-legend-swatch" style={{ background: item.color }} aria-hidden="true" />
                <span>{item.label}</span>
                <strong>{item.value}</strong>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="card overview-chart-card overview-rate-card">
        <h3>{t('confirmRate')}</h3>
        <div className="overview-rate-body">
          <div className="overview-rate-chart">
            <svg
              viewBox="0 0 120 120"
              role="img"
              aria-label={`${t('confirmRate')}: ${confirmationRate}%`}
            >
              <circle
                className="overview-rate-track"
                cx="60"
                cy="60"
                r={DONUT_RADIUS}
                strokeWidth={DONUT_STROKE_WIDTH}
              />
              <circle
                key={confirmationRate}
                className="overview-rate-progress"
                cx="60"
                cy="60"
                r={DONUT_RADIUS}
                strokeWidth={DONUT_STROKE_WIDTH}
                pathLength="100"
                strokeDasharray="100"
                strokeDashoffset={100 - confirmationRate}
                strokeLinecap="butt"
                transform="rotate(-90 60 60)"
              />
            </svg>
          </div>
          <div className="overview-rate-copy">
            <strong>{confirmationRate}%</strong>
            <span>
              {t('confirmRateSummary')
                .replace('{confirmed}', confirmedRateRecords)
                .replace('{total}', decidedRecords)}
            </span>
          </div>
        </div>
      </section>
    </div>
  )
}

function formatSignedScore(value) {
  const rounded = Math.round((Number(value) || 0) * 100) / 100
  if (Object.is(rounded, -0) || rounded === 0) return '0'
  return rounded > 0 ? `+${rounded}` : String(rounded)
}

function TeamTargetSummary({ overview, fmtMonth, t }) {
  const actualScore = Number(overview.actual_score) || 0
  const targetScore = Number(overview.target_score) || 0
  const gap = Math.round((actualScore - targetScore) * 100) / 100
  const hasTarget = targetScore > 0
  const direction = !hasTarget ? 'unavailable' : gap > 0 ? 'up' : gap < 0 ? 'down' : 'equal'
  const comparisonLabel = !hasTarget
    ? t('teamTargetUnavailable')
    : gap > 0
      ? t('teamTargetExceeded')
      : gap < 0
        ? t('teamTargetBehind')
        : t('teamTargetMet')

  return (
    <section className={`team-performance-summary team-performance-${direction}`}>
      <div>
        <strong>{t('teamTargetProgress')}</strong>
        <p>
          {fmtMonth(overview.month)}: {t('teamScored')}{' '}
          <b>{actualScore} {t('points')}</b>.
        </p>
      </div>
      <div className="team-performance-comparison">
        <strong>{hasTarget ? `${formatSignedScore(gap)} ${t('points')}` : '—'}</strong>
        <span>{comparisonLabel}</span>
        <span className="team-performance-baseline">
          {t('monthlyNorm')}: {hasTarget ? `${targetScore} ${t('points')}` : '—'}
        </span>
        <span className="team-performance-baseline">
          {t('teamOnTarget')}: {overview.on_target_count} / {overview.officers_count}
        </span>
      </div>
    </section>
  )
}

function OfficerProgressChart({ comparisons, fmtMonth, t }) {
  return (
    <div className="card chart-card chart-wide officer-progress-chart">
      <h3>{t('personalProgress')}</h3>
      {comparisons.history.length ? (
        <ResponsiveContainer width="100%" height={230}>
          <LineChart data={comparisons.history} margin={{ top: 12, right: 24, bottom: 8, left: 0 }}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="month" tickFormatter={fmtMonth} fontSize={11} />
            <YAxis domain={['auto', 'auto']} />
            <Tooltip
              labelFormatter={fmtMonth}
              formatter={(value) => [value, t('points')]}
            />
            <Line
              type="monotone"
              dataKey="score"
              name={t('confirmedFinal')}
              stroke="#1e4d8c"
              strokeWidth={3}
              dot={{ r: 4, fill: '#fff', strokeWidth: 2 }}
              activeDot={{ r: 5 }}
            />
          </LineChart>
        </ResponsiveContainer>
      ) : (
        <div className="chart-empty">{t('nothingFound')}</div>
      )}
    </div>
  )
}

function EffectivenessBadge({ category, t }) {
  const labels = {
    very_effective: t('categoryVeryEffective'),
    effective: t('categoryEffective'),
    little_effective: t('categoryLittleEffective'),
  }
  return (
    <span className={`effectiveness-badge effectiveness-${category}`}>
      {labels[category]}
    </span>
  )
}

function KpiBlockProgress({ blocks, t, className = '' }) {
  if (!blocks?.length) return null
  return (
    <section className={`kpi-block-progress-section ${className}`.trim()}>
      <div className="section-head">
        <h2>{t('targetBlockOverview')}</h2>
      </div>
      <div className="card kpi-block-progress">
        {blocks.map((item) => {
          const percent = Math.min(100, Math.max(0, Number(item.achievement_percent) || 0))
          return (
            <div className="kpi-block-progress-item" key={item.block}>
              <div className="kpi-block-progress-head">
                <strong>{item.block}</strong>
                <span>{percent}%</span>
              </div>
              <div
                className="kpi-block-progress-track"
                role="progressbar"
                aria-label={`${item.block}: ${percent}%`}
                aria-valuemin="0"
                aria-valuemax="100"
                aria-valuenow={percent}
              >
                <div
                  className={`kpi-block-progress-fill progress-${effectivenessCategory(percent)}`}
                  style={{ width: `${percent}%` }}
                />
              </div>
              <div className="kpi-block-progress-detail">
                {t('blockScoreProgress')
                  .replace('{actual}', item.actual_score)
                  .replace('{target}', item.target_score)}
              </div>
            </div>
          )
        })}
      </div>
    </section>
  )
}

function TargetOfficerToolbar({ category, month, district, onCategory, t }) {
  const params = new URLSearchParams()
  if (category) params.set('category', category)
  if (month) params.set('month', month)
  if (district) params.set('district', district)
  const query = params.toString()
  const fullListUrl = `/officers${query ? `?${query}` : ''}`

  return (
    <div className="target-officer-toolbar">
      <select
        className="select"
        value={category}
        onChange={(event) => onCategory(event.target.value)}
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
      <Link className="full-list-link" to={fullListUrl}>{t('seeFullList')} →</Link>
    </div>
  )
}

function PenaltyPanel({ score, count, t }) {
  return (
    <aside className="target-penalty-panel" aria-label={t('penaltyPanel')}>
      <div>
        <strong>{t('penaltyPanel')}</strong>
        <span>{t('penaltyPanelHint')}</span>
      </div>
      <dl>
        <div>
          <dt>{t('penaltyRecords')}</dt>
          <dd>{count || 0}</dd>
        </div>
        <div>
          <dt>{t('penaltyPoints')}</dt>
          <dd>{Number(score) || 0}</dd>
        </div>
      </dl>
    </aside>
  )
}

function ImpactPanel({ className = '', title, hint, items }) {
  return (
    <section className={`dashboard-impact-panel ${className}`.trim()}>
      <div className="dashboard-impact-copy">
        <h3>{title}</h3>
        <p>{hint}</p>
      </div>
      <dl>
        {items.map((item) => (
          <div key={item.label}>
            <dt>{item.label}</dt>
            <dd>{item.value}</dd>
          </div>
        ))}
      </dl>
    </section>
  )
}

function ExceededLimitsPanel({ items, t, lang }) {
  return (
    <section className="card chart-card exceeded-limits-card">
      <div className="exceeded-limits-head">
        <div>
          <h3>{t('mostExceededLimits')}</h3>
          <p>{t('mostExceededLimitsHint')}</p>
        </div>
      </div>
      {items.length ? (
        <div className="exceeded-limits-scroll">
          <table className="data-table exceeded-limits-table">
            <thead>
              <tr>
                <th>{t('colCode')}</th>
                <th>{t('colIndicator')}</th>
                <th className="num">{t('reportedIncidents')}</th>
                <th className="num">{t('countedIncidents')}</th>
                <th className="num">{t('excludedByLimits')}</th>
                <th className="num">{t('officersAffected')}</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr key={item.code}>
                  <td><span className="code-chip">{item.code}</span></td>
                  <td>{kpiIndicator(item, lang)}</td>
                  <td className="num">{item.reported}</td>
                  <td className="num">{item.counted}</td>
                  <td className="num limit-excluded-value">{item.excluded}</td>
                  <td className="num">{item.affected_officers}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="exceeded-limits-empty">{t('noLimitExcess')}</div>
      )}
    </section>
  )
}

export default function Dashboard({ adminMode = false }) {
  const { t, fmtMonth, lang } = useI18n()
  const { isHead } = useAuth()
  const navigate = useNavigate()
  const { officerId = '' } = useParams()
  const [data, setData] = useState(null)
  const [comparisons, setComparisons] = useState(null)
  const [targetOverview, setTargetOverview] = useState(null)
  const [months, setMonths] = useState([])
  const [month, setMonth] = useState('')
  const [officers, setOfficers] = useState([])
  const [officer, setOfficer] = useState(officerId)
  const [districts, setDistricts] = useState([])
  const [district, setDistrict] = useState('')
  const [blocks, setBlocks] = useState([])
  const [block, setBlock] = useState('')
  const [targetCategory, setTargetCategory] = useState('')
  const [ready, setReady] = useState(false)
  const [loadError, setLoadError] = useState('')
  const compactBlockChart = useCompactChartLayout()
  const adminEndpoints = adminMode ? adminOfficerDashboardEndpoints(officerId) : null
  const officerOptions = useMemo(
    () => officers
      .filter((item) => !district || String(item.districtId) === String(district))
      .map((item) => ({
      id: item.officer_id,
      label: item.name,
      description: [item.rank, districtName(item.district, lang)].filter(Boolean).join(' · '),
    })),
    [district, lang, officers]
  )

  const load = async (m, oid, did, availableMonths = months) => {
    const params = {}
    if (m) params.month = m
    if (oid) params.officer_id = oid
    if (did) params.district_id = did
    const targetParams = { ...params }
    const targetMonth = m || availableMonths[0] || ''
    if (targetMonth) targetParams.month = targetMonth
    if (adminMode && !adminEndpoints) {
      setLoadError(t('officerDashboardNotFound'))
      return
    }
    try {
      setLoadError('')
      const [dashboardResponse, targetOverviewResponse] = await Promise.all([
        api.get(adminMode ? adminEndpoints.dashboard : '/dashboard', { params: adminMode ? { month: m } : params }),
        api.get(adminMode ? adminEndpoints.targetOverview : '/target-overview', {
          params: adminMode ? { month: targetMonth } : targetParams,
        }),
      ])
      const dashboardData = dashboardResponse.data
      setData(dashboardData)
      setComparisons(dashboardData.comparisons)
      setBlocks((dashboardData.blocks || []).map((item) => item.block).filter(Boolean))
      setTargetOverview(targetOverviewResponse.data)
    } catch (err) {
      setLoadError(getApiErrorMessage(err, t, 'dashboardLoadError'))
    }
  }

  useEffect(() => {
    Promise.all([
      api.get('/months'),
      api.get('/districts'),
      isHead || adminMode ? api.get('/officers') : Promise.resolve({ data: [] }),
    ]).then(([monthsResponse, districtsResponse, officersResponse]) => {
      const monthItems = Array.isArray(monthsResponse.data) ? monthsResponse.data : []
      setMonths(monthItems)
      const initialMonth = monthItems[0] || ''
      setMonth(initialMonth)
      setDistricts(Array.isArray(districtsResponse.data) ? districtsResponse.data : [])
      const officerItems = Array.isArray(officersResponse.data) ? officersResponse.data : []
      const directory = officerItems.map((item) => ({
        officer_id: item.id,
        name: item.full_name,
        rank: item.rank,
        district: item.district?.name || '',
        districtId: item.district?.id || '',
        efficiencyRank: item.efficiency_rank,
        efficiencyTotal: item.efficiency_total,
        totalScore: item.efficiency_score,
      }))
      setOfficers(directory)
      return load(initialMonth, officerId, '', monthItems)
    }).catch((err) => {
      setLoadError(getApiErrorMessage(err, t, 'dashboardLoadError'))
    }).finally(() => setReady(true))
  }, [])

  useEffect(() => {
    if (!ready || String(officer) === String(officerId)) return
    setOfficer(officerId)
    setDistrict('')
    setBlock('')
    load(month, officerId, '')
  }, [officerId, ready])

  const onMonth = (m) => {
    setMonth(m)
    load(m, officer, district)
  }

  const onOfficer = (oid) => {
    setOfficer(oid)
    setDistrict('')
    setBlock('')
    navigate(oid ? `/dashboard/officers/${oid}` : '/dashboard')
    load(month, oid, '')
  }

  const onDistrict = (did) => {
    setDistrict(did)
    setOfficer('')
    setBlock('')
    navigate('/dashboard')
    load(month, '', did)
  }

  const openOfficerDashboard = (oid) => {
    const selectedId = String(oid)
    setOfficer(selectedId)
    setDistrict('')
    setBlock('')
    navigate(`/dashboard/officers/${selectedId}`)
    load(month, selectedId, '')
    window.scrollTo({ top: 0 })
  }

  if (!data && loadError) {
    return (
      <div className="card dashboard-error-state">
        <div className="error">{loadError}</div>
        {adminMode && <Link className="btn btn-ghost" to="/admin">{t('backToEmployees')}</Link>}
      </div>
    )
  }
  if (!data) return <div className="loading">Загрузка…</div>

  const statusTrend = data.status_trend || []
  const indicatorVolume = data.indicator_volume || []
  const exceededLimits = data.exceeded_limits || []
  const monthlyLimitImpact = data.monthly_limit_impact || {}
  const criticalIndicatorImpact = data.critical_indicator_impact || {}
  const inactiveOfficers = data.inactive_officers || { count: 0, total: 0 }
  const districtReview = (data.district_review || []).map((item) => ({
    ...item,
    district: districtName(item.district, lang),
  }))
  const targetKpiItems = officer
    ? (targetOverview?.items || []).filter((item) => Number(item.confirmed_quantity) > 0)
    : []
  const targetOfficerAverageScore = averageEffectivenessScore(targetOverview?.officers || [])
  const targetOfficerRows = !officer
    ? (targetOverview?.officers || [])
      .map((item) => ({
        ...item,
        achievement_percent: relativeScorePercent(item.actual_score, targetOfficerAverageScore),
        effectivenessCategory: effectivenessCategory(item, targetOfficerAverageScore),
      }))
      .filter((item) => (
        !targetCategory || item.effectivenessCategory === targetCategory
      ))
      .sort(compareEffectiveness)
      .slice(0, 10)
    : []
  const selectedOfficer = officers.find((item) => (
    String(item.officer_id) === String(officer)
  ))
  const selectedDistrict = districts.find((item) => (
    String(item.id) === String(district)
  ))

  // "Эффективность по сотрудникам": при выборе блока считаем баллы только по нему
  // и показываем сотрудников с любым ненулевым результатом, включая штрафные баллы.
  let officerChart = data.officers.map((o) => ({
    name: o.name.split(' ').slice(0, 2).join(' '),
    final: block ? (o.blocks?.[block] || 0) : o.final,
  }))
  if (block) {
    officerChart = officerChart.filter((o) => o.final !== 0).sort((a, b) => b.final - a.final)
  }
  const blockChartRows = blockScoreRows(data, officer)

  return (
    <div>
      <div className="page-head">
        <h1>
          {selectedOfficer
            ? `${t('dashboard')}: ${selectedOfficer.name}`
            : selectedDistrict
              ? `${t('dashboard')}: ${districtName(selectedDistrict, lang)}`
              : t('dashboard')}
        </h1>
        <div className="page-head-actions">
          {adminMode && <Link className="btn btn-ghost" to="/admin">{t('backToEmployees')}</Link>}
          {isHead && (
            <select
              className="select dashboard-district-select"
              value={district}
              onChange={(event) => onDistrict(event.target.value)}
              aria-label={t('selectGroup')}
            >
              <option value="">{t('allGroups')}</option>
              {districts.map((item) => (
                <option key={item.id} value={item.id}>{districtName(item, lang)}</option>
              ))}
            </select>
          )}
          {isHead && (
            <NameFilterCombobox
              items={officerOptions}
              value={officer}
              onChange={onOfficer}
            />
          )}
          <select className="select" value={month} onChange={(e) => onMonth(e.target.value)}>
            <option value="">{t('allMonths')}</option>
            {months.map((m) => <option key={m} value={m}>{fmtMonth(m)}</option>)}
          </select>
        </div>
      </div>

      {!officer && targetOverview && (
        <TeamTargetSummary overview={targetOverview} fmtMonth={fmtMonth} t={t} />
      )}

      <div className="section-head">
        <h2>{t('overviewMetrics')}</h2>
      </div>
      <OverviewCharts data={data} t={t} />

      {targetOverview && (
        <>
          <div className="section-head">
            <h2>{t('targetOverview')}</h2>
            <div className="section-note">{t('summaryForMonth')} {fmtMonth(targetOverview.month)}</div>
          </div>
          <div className="stat-grid">
            <StatCard label={t('actualScore')} value={targetOverview.actual_score} accent="#1e4d8c" />
            <StatCard label={t('monthlyNorm')} value={targetOverview.target_score} accent="#5b4b8a" />
            {!officer && (
              <StatCard
                label={t('officersOnTarget')}
                value={`${targetOverview.on_target_count} / ${targetOverview.officers_count}`}
                accent="#b8860b"
              />
            )}
            {!officer && (
              <StatCard
                label={t('employeesWithoutActivity')}
                value={`${inactiveOfficers.count} / ${inactiveOfficers.total}`}
                accent="#9c3848"
              />
            )}
          </div>
          <div className="dashboard-impact-grid">
            <ImpactPanel
              className="monthly-limit-impact"
              title={t('monthlyLimitImpact')}
              hint={t('monthlyLimitImpactHint')}
              items={[
                { label: t('reportedIncidents'), value: monthlyLimitImpact.reported || 0 },
                { label: t('countedIncidents'), value: monthlyLimitImpact.counted || 0 },
                { label: t('excludedByLimits'), value: monthlyLimitImpact.excluded || 0 },
                { label: t('recordsOverLimit'), value: monthlyLimitImpact.affected_records || 0 },
              ]}
            />
            <ImpactPanel
              className="critical-indicator-impact"
              title={t('criticalIndicatorImpact')}
              hint={t('criticalIndicatorImpactHint')}
              items={[
                { label: t('criticalRecords'), value: criticalIndicatorImpact.records || 0 },
                { label: t('criticalOfficers'), value: criticalIndicatorImpact.officers || 0 },
                { label: t('criticalIncidents'), value: criticalIndicatorImpact.quantity || 0 },
              ]}
            />
          </div>
        </>
      )}

      {targetOverview && !officer && (
        <EffectivenessOverviewChart officers={targetOverview.officers || []} t={t} />
      )}

      {targetOverview && officer && (
        <KpiBlockProgress
          blocks={targetOverview.blocks}
          t={t}
          className="kpi-block-progress-personal"
        />
      )}

      {targetOverview && (
        <div className="card table-card target-overview-card">
          <div className="chart-head target-overview-head">
            <h3>{officer ? t('targetKpiBreakdown') : t('targetOfficerOverview')}</h3>
            {!officer && (
              <TargetOfficerToolbar
                category={targetCategory}
                month={targetOverview.month}
                district={district}
                onCategory={setTargetCategory}
                t={t}
              />
            )}
          </div>
          {!officer && (
            <PenaltyPanel
              score={targetOverview.penalty_score}
              count={targetOverview.penalty_count}
              t={t}
            />
          )}
          {!officer && <div className="target-overview-note">{t('topTenOfficers')}</div>}
          <div className="table-scroll">
            <table className="data-table">
              <thead>
                {officer ? (
                  <tr>
                    <th>{t('colCode')}</th>
                    <th>{t('colIndicator')}</th>
                    <th className="num">{t('colActualQuantity')}</th>
                    <th className="num">{t('colCountedQuantity')}</th>
                    <th className="num">{t('colActualScore')}</th>
                    <th className="num">{t('monthlyLimit')}</th>
                    <th>{t('massIndicator')}</th>
                    <th>{t('criticalIndicator')}</th>
                  </tr>
                ) : (
                  <tr>
                    <th>{t('colOfficer')}</th>
                    <th className="num">{t('actualScore')}</th>
                    <th className="num">{t('monthlyNorm')}</th>
                    <th className="num">{t('colAchievement')}</th>
                    <th className="num">{t('colGap')}</th>
                    <th className="num">{t('penaltyPoints')}</th>
                    <th>{t('effectivenessCategory')}</th>
                  </tr>
                )}
              </thead>
              <tbody>
                {officer ? targetKpiItems.map((item) => (
                  <tr key={item.catalog_id}>
                    <td><span className="code-chip">{item.code}</span></td>
                    <td>{kpiIndicator(item, lang)}</td>
                    <td className="num">{item.confirmed_quantity}</td>
                    <td className="num">{item.counted_quantity}</td>
                    <td className="num strong">{item.confirmed_score}</td>
                    <td className="num">{item.monthly_limit}</td>
                    <td>{item.is_mass ? t('yes') : t('no')}</td>
                    <td>{item.is_critical ? t('yes') : t('no')}</td>
                  </tr>
                )) : targetOfficerRows.map((item) => {
                  const directoryOfficer = officers.find((entry) => (
                    String(entry.officer_id) === String(item.officer_id)
                  ))
                  return (
                    <tr key={item.officer_id}>
                      <td className="nowrap">
                        <ProfileHoverCard profile={{
                          id: item.officer_id,
                          name: item.name,
                          rank: directoryOfficer?.rank,
                          district: directoryOfficer?.district,
                          efficiencyRank: directoryOfficer?.efficiencyRank,
                          efficiencyTotal: directoryOfficer?.efficiencyTotal,
                          totalScore: directoryOfficer?.totalScore,
                        }}>
                          <button
                            type="button"
                            className="officer-dashboard-link"
                            onClick={() => openOfficerDashboard(item.officer_id)}
                            aria-label={`${t('openOfficerDashboard')}: ${item.name}`}
                            title={t('openOfficerDashboard')}
                          >
                            {item.name}
                          </button>
                        </ProfileHoverCard>
                      </td>
                      <td className="num strong">{item.actual_score}</td>
                      <td className="num">{item.target_score}</td>
                      <td className="num">{item.achievement_percent}%</td>
                      <td className={`num ${item.gap < 0 ? 'target-gap-negative' : 'target-gap-positive'}`}>
                        {item.gap > 0 ? `+${item.gap}` : item.gap}
                      </td>
                      <td className={`num ${item.penalty_score < 0 ? 'penalty-score' : ''}`}>
                        {item.penalty_score || 0}
                      </td>
                      <td>
                        <EffectivenessBadge category={item.effectivenessCategory} t={t} />
                      </td>
                    </tr>
                  )
                })}
                {((officer && targetKpiItems.length === 0) || (!officer && targetOfficerRows.length === 0)) && (
                  <tr><td colSpan={officer ? 7 : 7} className="empty">{t('nothingFound')}</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {targetOverview && (
        <ExceededLimitsPanel items={exceededLimits} t={t} lang={lang} />
      )}

      {officer && !month && comparisons && (
        <div className="chart-grid comparison-grid dashboard-chart-grid">
          <OfficerProgressChart comparisons={comparisons} fmtMonth={fmtMonth} t={t} />
        </div>
      )}

      <div className="chart-grid dashboard-chart-grid">
        {!officer && <>
          <div className="card chart-card chart-wide">
          <div className="chart-head">
            <h3>{t('byOfficer')}</h3>
            <select className="select" value={block} onChange={(e) => setBlock(e.target.value)}>
              <option value="">{t('allBlocks')}</option>
              {blocks.map((b) => <option key={b} value={b}>{b}</option>)}
            </select>
          </div>
          <ResponsiveContainer width="100%" height={230}>
            <BarChart data={officerChart} margin={{ top: 10, right: 20, bottom: 40, left: 0 }}>
              <CartesianGrid strokeDasharray="3 3" vertical={false} />
              <XAxis dataKey="name" angle={-20} textAnchor="end" height={60} fontSize={12} />
              <YAxis domain={['auto', 'auto']} />
              <Tooltip cursor={CHART_CURSOR} />
              <ReferenceLine y={0} stroke="#6b7785" />
              <Bar dataKey="final" name={t('colFinal')} radius={[4, 4, 0, 0]} maxBarSize={block ? 64 : 28}>
                {officerChart.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>

        {targetOverview && (
          <KpiBlockProgress blocks={targetOverview.blocks} t={t} />
        )}

        {!month && (
          <div className="card chart-card chart-wide">
            <h3>{t('trend')}</h3>
            <ResponsiveContainer width="100%" height={200}>
              <LineChart data={data.trend} margin={{ top: 10, right: 20, bottom: 10, left: 0 }}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="month" fontSize={12} tickFormatter={fmtMonth} />
                <YAxis />
                <Tooltip cursor={CHART_CURSOR} labelFormatter={fmtMonth} />
                <Line type="monotone" dataKey="final" name={t('colFinal')} stroke="#1e4d8c" strokeWidth={3} dot={{ r: 4 }} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        )}

        </>}

        <div className="card chart-card chart-wide block-chart-card">
          <h3>{t('byBlock')}</h3>
          <ResponsiveContainer width="100%" height={Math.min(280, Math.max(210, blockChartRows.length * 30))}>
            <BarChart
              data={blockChartRows}
              layout="vertical"
              margin={{ top: 5, right: compactBlockChart ? 10 : 30, bottom: 5, left: 0 }}
            >
              <CartesianGrid strokeDasharray="3 3" horizontal={false} />
              <XAxis type="number" domain={blockScoreDomain} />
              <YAxis
                type="category"
                dataKey="block"
                width={compactBlockChart ? 112 : 220}
                tick={<BlockAxisTick maxLength={compactBlockChart ? 14 : null} />}
                tickLine={false}
              />
              <Tooltip cursor={CHART_CURSOR} formatter={(value) => [value, t('points')]} />
              <ReferenceLine x={0} stroke="#6b7785" />
              <Bar dataKey="final" name={t('colFinal')} radius={[0, 4, 4, 0]} maxBarSize={28}>
                {blockChartRows.map((item, i) => (
                  <Cell
                    key={item.block}
                    fill={item.final < 0 ? STATUS_COLORS.rejected : BLOCK_COLORS[i % BLOCK_COLORS.length]}
                  />
                ))}
                <LabelList
                  dataKey="final"
                  content={<BlockScoreLabel pointsLabel={t('points')} />}
                />
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>

        {!month && (
          <div className="card chart-card chart-wide">
            <h3>{t('reviewFlow')}</h3>
            <ResponsiveContainer width="100%" height={210}>
              <BarChart data={statusTrend} margin={{ top: 10, right: 20, bottom: 10, left: 0 }}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="month" tickFormatter={fmtMonth} fontSize={12} />
                <YAxis allowDecimals={false} />
                <Tooltip cursor={CHART_CURSOR} labelFormatter={fmtMonth} />
                <Legend />
                <Bar dataKey="confirmed" name={t('confirmed')} stackId="status" fill={STATUS_COLORS.confirmed} barSize={12} />
                <Bar dataKey="pending" name={t('pending')} stackId="status" fill={STATUS_COLORS.pending} barSize={12} />
                <Bar dataKey="rejected" name={t('rejected')} stackId="status" fill={STATUS_COLORS.rejected} radius={[3, 3, 0, 0]} barSize={12} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}

        <div className="card chart-card indicator-volume-card">
          <h3>{t('indicatorVolume')}</h3>
          <ResponsiveContainer width="100%" height={220}>
            <BarChart data={indicatorVolume} layout="vertical" margin={{ top: 5, right: 25, bottom: 5, left: 10 }}>
              <CartesianGrid strokeDasharray="3 3" horizontal={false} />
              <XAxis type="number" />
              <YAxis type="category" dataKey="code" width={58} fontSize={12} />
              <Tooltip
                content={<IndicatorVolumeTooltip t={t} lang={lang} />}
                allowEscapeViewBox={{ x: false, y: true }}
                offset={12}
                cursor={CHART_CURSOR}
                wrapperStyle={{ zIndex: 5 }}
              />
              <Bar dataKey="quantity" name={t('unitsCompleted')} fill="#0f8b8d" radius={[0, 4, 4, 0]} barSize={12} />
            </BarChart>
          </ResponsiveContainer>
        </div>

        <div className="card chart-card">
          <h3>{t('districtReview')}</h3>
          <ResponsiveContainer width="100%" height={220}>
            <BarChart data={districtReview} layout="vertical" margin={{ top: 5, right: 25, bottom: 5, left: 15 }}>
              <CartesianGrid strokeDasharray="3 3" horizontal={false} />
              <XAxis type="number" allowDecimals={false} />
              <YAxis type="category" dataKey="district" width={125} fontSize={11} />
              <Tooltip cursor={CHART_CURSOR} />
              <Legend />
              <Bar dataKey="pending" name={t('pending')} stackId="review" fill={STATUS_COLORS.pending} barSize={12} />
              <Bar dataKey="rejected" name={t('rejected')} stackId="review" fill={STATUS_COLORS.rejected} radius={[0, 4, 4, 0]} barSize={12} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  )
}
