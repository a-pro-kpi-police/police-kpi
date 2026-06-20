import { useEffect, useState } from 'react'
import {
  Bar, BarChart, CartesianGrid, Cell, Legend, Line, LineChart, Pie, PieChart,
  ResponsiveContainer, Tooltip, XAxis, YAxis,
} from 'recharts'
import api from '../api'
import { useI18n } from '../i18n.jsx'

const COLORS = ['#1e4d8c', '#2f7d4f', '#b8860b', '#9c3848', '#5b4b8a', '#0f8b8d', '#a85b2b', '#3b6ea5']

function StatCard({ label, value, accent }) {
  return (
    <div className="stat-card" style={{ borderTopColor: accent }}>
      <div className="stat-value">{value}</div>
      <div className="stat-label">{label}</div>
    </div>
  )
}

export default function Dashboard() {
  const { t, fmtMonth } = useI18n()
  const [data, setData] = useState(null)
  const [months, setMonths] = useState([])
  const [month, setMonth] = useState('')
  const [blocks, setBlocks] = useState([])
  const [block, setBlock] = useState('')

  const load = async (m) => {
    const { data } = await api.get('/dashboard', { params: m ? { month: m } : {} })
    setData(data)
  }

  useEffect(() => {
    api.get('/months').then((r) => setMonths(r.data))
    api.get('/catalog').then((r) => {
      const list = []
      for (const c of r.data) if (c.block && !list.includes(c.block)) list.push(c.block)
      setBlocks(list)
    })
    load('')
  }, [])

  const onMonth = (m) => {
    setMonth(m)
    load(m)
  }

  if (!data) return <div className="loading">Загрузка…</div>

  // "Эффективность по сотрудникам": при выборе блока считаем баллы только по нему
  // и показываем лишь сотрудников с баллами по блоку — столбцы становятся крупнее.
  let officerChart = data.officers.map((o) => ({
    name: o.name.split(' ').slice(0, 2).join(' '),
    final: block ? (o.blocks?.[block] || 0) : o.final,
  }))
  if (block) {
    officerChart = officerChart.filter((o) => o.final > 0).sort((a, b) => b.final - a.final)
  }

  return (
    <div>
      <div className="page-head">
        <h1>{t('dashboard')}</h1>
        <select className="select" value={month} onChange={(e) => onMonth(e.target.value)}>
          <option value="">{t('allMonths')}</option>
          {months.map((m) => <option key={m} value={m}>{fmtMonth(m)}</option>)}
        </select>
      </div>

      <div className="stat-grid">
        <StatCard label={t('totalFinal')} value={data.total_final} accent="#1e4d8c" />
        <StatCard label={t('confirmedCount')} value={data.confirmed_count} accent="#2f7d4f" />
        <StatCard label={t('pendingCount')} value={data.pending_count} accent="#b8860b" />
        <StatCard label={t('rejectedCount')} value={data.rejected_count} accent="#9c3848" />
        <StatCard label={t('officersCount')} value={data.officers_count} accent="#5b4b8a" />
        <StatCard label={t('confirmRate')} value={`${data.confirm_rate}%`} accent="#0f8b8d" />
      </div>

      <div className="chart-grid">
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
              <YAxis />
              <Tooltip />
              <Bar dataKey="final" name={t('colFinal')} radius={[4, 4, 0, 0]} maxBarSize={block ? 64 : 28}>
                {officerChart.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>

        <div className="card chart-card chart-wide">
          <h3>{t('byBlock')}</h3>
          <ResponsiveContainer width="100%" height={340}>
            <PieChart>
              <Pie data={data.blocks} dataKey="final" nameKey="block" cx="50%" cy="45%" outerRadius={100} label>
                {data.blocks.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
              </Pie>
              <Tooltip />
              <Legend
                verticalAlign="bottom"
                layout="horizontal"
                wrapperStyle={{ fontSize: 12, paddingTop: 8 }}
              />
            </PieChart>
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
      </div>
    </div>
  )
}
