import { useEffect, useState } from 'react'
import {
  Bar, BarChart, CartesianGrid, Cell, Legend, Line, LineChart, ReferenceLine,
  ResponsiveContainer, Tooltip, XAxis, YAxis,
} from 'recharts'
import api from '../api'
import { useAuth } from '../auth.jsx'
import { useI18n } from '../i18n.jsx'

const COLORS = ['#1e4d8c', '#2f7d4f', '#b8860b', '#9c3848', '#5b4b8a', '#0f8b8d', '#a85b2b', '#3b6ea5']
const BLOCK_COLORS = ['#1e4d8c', '#2f7d4f', '#b8860b', '#5b4b8a', '#0f8b8d', '#a85b2b', '#3b6ea5']
const STATUS_COLORS = { confirmed: '#2f7d4f', pending: '#b8860b', rejected: '#9c3848' }

function median(values) {
  if (!values.length) return 0
  const sorted = [...values].sort((a, b) => a - b)
  const middle = Math.floor(sorted.length / 2)
  return sorted.length % 2 ? sorted[middle] : (sorted[middle - 1] + sorted[middle]) / 2
}

function buildComparisons(entries, selectedMonth, officerId) {
  const periodEntries = selectedMonth ? entries.filter((entry) => entry.month === selectedMonth) : entries
  const byOfficer = {}

  for (const entry of periodEntries) {
    if (!byOfficer[entry.officer_id]) {
      byOfficer[entry.officer_id] = {
        officer_id: entry.officer_id,
        name: entry.officer_name,
        score: 0,
      }
    }
    const item = byOfficer[entry.officer_id]
    if (entry.status === 'confirmed') {
      item.score += entry.final_score
    }
  }

  const officers = Object.values(byOfficer)
  const peerMedian = median(officers.map((item) => item.score))
  const selected = officers.find((item) => String(item.officer_id) === String(officerId))
  const focusScore = officerId
    ? (selected?.score || 0)
    : (officers.length ? officers.reduce((sum, item) => sum + item.score, 0) / officers.length : 0)

  const focusHistory = officerId
    ? entries.filter((entry) => String(entry.officer_id) === String(officerId))
    : entries
  const currentMonth = selectedMonth || [...new Set(focusHistory.map((entry) => entry.month))].sort().at(-1) || ''
  const scoresByMonth = {}
  for (const entry of focusHistory) {
    if (entry.status !== 'confirmed') continue
    scoresByMonth[entry.month] = (scoresByMonth[entry.month] || 0) + entry.final_score
  }
  const baselineValues = Object.entries(scoresByMonth)
    .filter(([month]) => month < currentMonth)
    .map(([, score]) => score)
  const currentScore = scoresByMonth[currentMonth] || 0
  const baseline = baselineValues.length
    ? baselineValues.reduce((sum, score) => sum + score, 0) / baselineValues.length
    : 0

  return {
    focusScore: Number(focusScore.toFixed(2)),
    peerMedian: Number(peerMedian.toFixed(2)),
    currentScore: Number(currentScore.toFixed(2)),
    baseline: Number(baseline.toFixed(2)),
    currentMonth,
  }
}

function buildSupplementalCharts(entries) {
  const months = {}
  const indicators = {}
  const reviews = {}

  for (const entry of entries) {
    if (!months[entry.month]) months[entry.month] = { confirmed: 0, pending: 0, rejected: 0 }
    if (entry.status in months[entry.month]) months[entry.month][entry.status] += 1

    const district = entry.district_name || '—'
    const code = entry.code || '—'
    if (!indicators[code]) indicators[code] = { code, indicator: entry.indicator || '—', quantity: 0 }
    indicators[code].quantity += entry.quantity

    if (entry.status === 'pending' || entry.status === 'rejected') {
      if (!reviews[district]) reviews[district] = { pending: 0, rejected: 0 }
      reviews[district][entry.status] += 1
    }
  }

  return {
    status_trend: Object.entries(months)
      .map(([month, counts]) => ({ month, ...counts }))
      .sort((a, b) => a.month.localeCompare(b.month)),
    indicator_volume: Object.values(indicators)
      .map((item) => ({ ...item, quantity: Number(item.quantity.toFixed(2)) }))
      .sort((a, b) => b.quantity - a.quantity),
    district_review: Object.entries(reviews)
      .map(([district, counts]) => ({ district, ...counts }))
      .sort((a, b) => (b.pending + b.rejected) - (a.pending + a.rejected)),
  }
}

function StatCard({ label, value, accent }) {
  return (
    <div className="stat-card" style={{ borderTopColor: accent }}>
      <div className="stat-value">{value}</div>
      <div className="stat-label">{label}</div>
    </div>
  )
}

function ComparisonCharts({ comparisons, fmtMonth, officer, t }) {
  return <>
    <div className="card chart-card">
      <h3>{t('peerMedian')}</h3>
      <ResponsiveContainer width="100%" height={220}>
        <BarChart data={[
          { label: officer ? t('selectedOfficer') : t('officerAverage'), value: comparisons.focusScore, color: '#1e4d8c' },
          { label: t('peerMedianValue'), value: comparisons.peerMedian, color: '#8ba3bd' },
        ]} margin={{ top: 10, right: 20, bottom: 15, left: 0 }}>
          <CartesianGrid strokeDasharray="3 3" vertical={false} />
          <XAxis dataKey="label" fontSize={11} />
          <YAxis />
          <Tooltip formatter={(value) => [value, t('points')]} />
          <Bar dataKey="value" radius={[4, 4, 0, 0]} maxBarSize={80}>
            {[0, 1].map((index) => <Cell key={index} fill={index === 0 ? '#1e4d8c' : '#8ba3bd'} />)}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>

    <div className="card chart-card">
      <h3>{t('personalBaseline')}</h3>
      <ResponsiveContainer width="100%" height={220}>
        <BarChart data={[
          { label: comparisons.currentMonth ? fmtMonth(comparisons.currentMonth) : t('currentMonth'), value: comparisons.currentScore },
          { label: t('personalBaselineValue'), value: comparisons.baseline },
        ]} margin={{ top: 10, right: 20, bottom: 15, left: 0 }}>
          <CartesianGrid strokeDasharray="3 3" vertical={false} />
          <XAxis dataKey="label" fontSize={11} />
          <YAxis />
          <Tooltip formatter={(value) => [value, t('points')]} />
          <Bar dataKey="value" fill="#0f8b8d" radius={[4, 4, 0, 0]} maxBarSize={80} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  </>
}

function TargetStatusBadge({ status, t }) {
  const classes = {
    on_target: 'badge-confirmed',
    behind: 'badge-rejected',
    no_target: 'badge-pending',
  }
  const labels = {
    on_target: t('targetStatusOnTarget'),
    behind: t('targetStatusBehind'),
    no_target: t('targetStatusNoTarget'),
  }
  return <span className={`badge ${classes[status] || 'badge-pending'}`}>{labels[status] || status}</span>
}

export default function Dashboard() {
  const { t, fmtMonth } = useI18n()
  const { isHead } = useAuth()
  const [data, setData] = useState(null)
  const [comparisons, setComparisons] = useState(null)
  const [targetOverview, setTargetOverview] = useState(null)
  const [months, setMonths] = useState([])
  const [month, setMonth] = useState('')
  const [officers, setOfficers] = useState([])
  const [officer, setOfficer] = useState('')
  const [blocks, setBlocks] = useState([])
  const [block, setBlock] = useState('')

  const load = async (m, oid) => {
    const params = {}
    if (m) params.month = m
    if (oid) params.officer_id = oid
    const [dashboardResponse, allEntriesResponse, targetOverviewResponse] = await Promise.all([
      api.get('/dashboard', { params }),
      api.get('/entries'),
      api.get('/target-overview', { params }),
    ])
    const filteredEntries = allEntriesResponse.data.filter((entry) =>
      (!m || entry.month === m) && (!oid || String(entry.officer_id) === String(oid)))
    setData({
      ...dashboardResponse.data,
      ...buildSupplementalCharts(filteredEntries),
    })
    setComparisons(buildComparisons(allEntriesResponse.data, m, oid))
    setTargetOverview(targetOverviewResponse.data)
  }

  useEffect(() => {
    api.get('/months').then((r) => setMonths(r.data))
    api.get('/catalog').then((r) => {
      const list = []
      for (const c of r.data) if (c.block && !list.includes(c.block)) list.push(c.block)
      setBlocks(list)
    })
    if (isHead) {
      api.get('/dashboard').then((r) => {
        setOfficers([...r.data.officers].sort((a, b) => a.name.localeCompare(b.name)))
      })
    }
    load('', '')
  }, [])

  const onMonth = (m) => {
    setMonth(m)
    load(m, officer)
  }

  const onOfficer = (oid) => {
    setOfficer(oid)
    setBlock('')
    load(month, oid)
  }

  if (!data) return <div className="loading">Загрузка…</div>

  const statusTrend = data.status_trend || []
  const indicatorVolume = data.indicator_volume || []
  const districtReview = data.district_review || []

  // "Эффективность по сотрудникам": при выборе блока считаем баллы только по нему
  // и показываем сотрудников с любым ненулевым результатом, включая штрафные баллы.
  let officerChart = data.officers.map((o) => ({
    name: o.name.split(' ').slice(0, 2).join(' '),
    final: block ? (o.blocks?.[block] || 0) : o.final,
  }))
  if (block) {
    officerChart = officerChart.filter((o) => o.final !== 0).sort((a, b) => b.final - a.final)
  }

  return (
    <div>
      <div className="page-head">
        <h1>{t('dashboard')}</h1>
        <div className="page-head-actions">
          {isHead && (
            <select className="select employee-select" value={officer} onChange={(e) => onOfficer(e.target.value)}>
              <option value="">{t('allOfficers')}</option>
              {officers.map((item) => (
                <option key={item.officer_id} value={item.officer_id}>{item.name}</option>
              ))}
            </select>
          )}
          <select className="select" value={month} onChange={(e) => onMonth(e.target.value)}>
            <option value="">{t('allMonths')}</option>
            {months.map((m) => <option key={m} value={m}>{fmtMonth(m)}</option>)}
          </select>
        </div>
      </div>

      <div className="section-head">
        <h2>{t('overviewMetrics')}</h2>
      </div>
      <div className="stat-grid">
        <StatCard label={t('confirmedCount')} value={data.confirmed_count} accent="#2f7d4f" />
        <StatCard label={t('pendingCount')} value={data.pending_count} accent="#b8860b" />
        <StatCard label={t('rejectedCount')} value={data.rejected_count} accent="#9c3848" />
        {!officer && <StatCard label={t('officersCount')} value={data.officers_count} accent="#5b4b8a" />}
        <StatCard label={t('confirmRate')} value={`${data.confirm_rate}%`} accent="#0f8b8d" />
      </div>

      {targetOverview && (
        <>
          <div className="section-head">
            <h2>{t('targetOverview')}</h2>
            <div className="section-note">{t('summaryForMonth')} {fmtMonth(targetOverview.month)}</div>
          </div>
          <div className="stat-grid">
            <StatCard label={t('actualScore')} value={targetOverview.actual_score} accent="#1e4d8c" />
            <StatCard label={t('targetScore')} value={targetOverview.target_score} accent="#5b4b8a" />
            <StatCard label={t('achievement')} value={`${targetOverview.achievement_percent}%`} accent="#2f7d4f" />
            {!officer && (
              <StatCard
                label={t('officersOnTarget')}
                value={`${targetOverview.on_target_count} / ${targetOverview.officers_count}`}
                accent="#b8860b"
              />
            )}
          </div>
        </>
      )}

      {targetOverview && (
        <div className="card table-card target-overview-card">
          <div className="chart-head target-overview-head">
            <h3>{officer ? t('targetKpiBreakdown') : t('targetOfficerOverview')}</h3>
          </div>
          <div className="table-scroll">
            <table className="data-table">
              <thead>
                {officer ? (
                  <tr>
                    <th>{t('colCode')}</th>
                    <th>{t('colIndicator')}</th>
                    <th className="num">{t('colActualQuantity')}</th>
                    <th className="num">{t('colTargetQuantity')}</th>
                    <th className="num">{t('colActualScore')}</th>
                    <th className="num">{t('colTargetScore')}</th>
                    <th className="num">{t('colAchievement')}</th>
                  </tr>
                ) : (
                  <tr>
                    <th>{t('colOfficer')}</th>
                    <th className="num">{t('actualScore')}</th>
                    <th className="num">{t('targetScore')}</th>
                    <th className="num">{t('colAchievement')}</th>
                    <th className="num">{t('colGap')}</th>
                    <th>{t('colStatus')}</th>
                  </tr>
                )}
              </thead>
              <tbody>
                {officer ? targetOverview.items.map((item) => (
                  <tr key={item.catalog_id}>
                    <td><span className="code-chip">{item.code}</span></td>
                    <td>{item.indicator}</td>
                    <td className="num">{item.confirmed_quantity}</td>
                    <td className="num">{item.target_quantity}</td>
                    <td className="num strong">{item.confirmed_score}</td>
                    <td className="num">{item.target_score}</td>
                    <td className="num">{item.target_score ? `${item.achievement_percent}%` : '—'}</td>
                  </tr>
                )) : targetOverview.officers.map((item) => (
                  <tr key={item.officer_id}>
                    <td className="nowrap strong">{item.name}</td>
                    <td className="num strong">{item.actual_score}</td>
                    <td className="num">{item.target_score}</td>
                    <td className="num">{item.target_score ? `${item.achievement_percent}%` : '—'}</td>
                    <td className={`num ${item.gap < 0 ? 'target-gap-negative' : 'target-gap-positive'}`}>
                      {item.gap > 0 ? `+${item.gap}` : item.gap}
                    </td>
                    <td><TargetStatusBadge status={item.status} t={t} /></td>
                  </tr>
                ))}
                {((officer && targetOverview.items.length === 0) || (!officer && targetOverview.officers.length === 0)) && (
                  <tr><td colSpan={officer ? 7 : 6} className="empty">{t('nothingFound')}</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {officer && comparisons && (
        <div className="chart-grid comparison-grid">
          <ComparisonCharts comparisons={comparisons} fmtMonth={fmtMonth} officer={officer} t={t} />
        </div>
      )}

      <div className="chart-grid">
        {!officer && <>
          <div className="card chart-card chart-wide">
          <div className="chart-head">
            <h3>{t('byOfficer')}</h3>
            <select className="select" value={block} onChange={(e) => setBlock(e.target.value)}>
              <option value="">{t('allBlocks')}</option>
              {blocks.map((b) => <option key={b} value={b}>{b}</option>)}
            </select>
          </div>
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={officerChart} margin={{ top: 10, right: 20, bottom: 40, left: 0 }}>
              <CartesianGrid strokeDasharray="3 3" vertical={false} />
              <XAxis dataKey="name" angle={-20} textAnchor="end" height={60} fontSize={12} />
              <YAxis domain={['auto', 'auto']} />
              <Tooltip />
              <ReferenceLine y={0} stroke="#6b7785" />
              <Bar dataKey="final" name={t('colFinal')} radius={[4, 4, 0, 0]} maxBarSize={block ? 64 : 28}>
                {officerChart.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>

        <div className="card chart-card chart-wide">
          <h3>{t('byBlock')}</h3>
          <ResponsiveContainer width="100%" height={Math.max(360, data.blocks.length * 46)}>
            <BarChart
              data={data.blocks}
              layout="vertical"
              margin={{ top: 5, right: 30, bottom: 5, left: 10 }}
            >
              <CartesianGrid strokeDasharray="3 3" horizontal={false} />
              <XAxis type="number" domain={['auto', 'auto']} />
              <YAxis type="category" dataKey="block" width={285} fontSize={11} />
              <Tooltip formatter={(value) => [value, t('points')]} />
              <ReferenceLine x={0} stroke="#6b7785" />
              <Bar dataKey="final" name={t('colFinal')} radius={[0, 4, 4, 0]} maxBarSize={28}>
                {data.blocks.map((item, i) => (
                  <Cell
                    key={item.block}
                    fill={item.final < 0 ? STATUS_COLORS.rejected : BLOCK_COLORS[i % BLOCK_COLORS.length]}
                  />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>

        <div className="card chart-card chart-wide">
          <h3>{t('trend')}</h3>
          <ResponsiveContainer width="100%" height={260}>
            <LineChart data={data.trend} margin={{ top: 10, right: 20, bottom: 10, left: 0 }}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="month" fontSize={12} tickFormatter={fmtMonth} />
              <YAxis />
              <Tooltip labelFormatter={fmtMonth} />
              <Line type="monotone" dataKey="final" name={t('colFinal')} stroke="#1e4d8c" strokeWidth={3} dot={{ r: 4 }} />
            </LineChart>
          </ResponsiveContainer>
        </div>

        {comparisons && (
          <ComparisonCharts comparisons={comparisons} fmtMonth={fmtMonth} officer={officer} t={t} />
        )}
        </>}

        <div className="card chart-card chart-wide">
          <h3>{t('reviewFlow')}</h3>
          <ResponsiveContainer width="100%" height={280}>
            <BarChart data={statusTrend} margin={{ top: 10, right: 20, bottom: 10, left: 0 }}>
              <CartesianGrid strokeDasharray="3 3" vertical={false} />
              <XAxis dataKey="month" tickFormatter={fmtMonth} fontSize={12} />
              <YAxis allowDecimals={false} />
              <Tooltip labelFormatter={fmtMonth} />
              <Legend />
              <Bar dataKey="confirmed" name={t('confirmed')} stackId="status" fill={STATUS_COLORS.confirmed} />
              <Bar dataKey="pending" name={t('pending')} stackId="status" fill={STATUS_COLORS.pending} />
              <Bar dataKey="rejected" name={t('rejected')} stackId="status" fill={STATUS_COLORS.rejected} radius={[3, 3, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>

        <div className="card chart-card">
          <h3>{t('indicatorVolume')}</h3>
          <ResponsiveContainer width="100%" height={320}>
            <BarChart data={indicatorVolume} layout="vertical" margin={{ top: 5, right: 25, bottom: 5, left: 10 }}>
              <CartesianGrid strokeDasharray="3 3" horizontal={false} />
              <XAxis type="number" />
              <YAxis type="category" dataKey="code" width={58} fontSize={12} />
              <Tooltip
                formatter={(value) => [value, t('unitsCompleted')]}
                labelFormatter={(code) => indicatorVolume.find((item) => item.code === code)?.indicator || code}
              />
              <Bar dataKey="quantity" name={t('unitsCompleted')} fill="#0f8b8d" radius={[0, 4, 4, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>

        <div className="card chart-card">
          <h3>{t('districtReview')}</h3>
          <ResponsiveContainer width="100%" height={320}>
            <BarChart data={districtReview} layout="vertical" margin={{ top: 5, right: 25, bottom: 5, left: 15 }}>
              <CartesianGrid strokeDasharray="3 3" horizontal={false} />
              <XAxis type="number" allowDecimals={false} />
              <YAxis type="category" dataKey="district" width={125} fontSize={11} />
              <Tooltip />
              <Legend />
              <Bar dataKey="pending" name={t('pending')} stackId="review" fill={STATUS_COLORS.pending} />
              <Bar dataKey="rejected" name={t('rejected')} stackId="review" fill={STATUS_COLORS.rejected} radius={[0, 4, 4, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  )
}
