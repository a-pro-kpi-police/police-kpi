const LABEL_EDGE_PADDING = 6

export function formatBlockScore(value, pointsLabel) {
  return `${value ?? 0} ${pointsLabel}`
}

export function blockScoreRows(dashboard, officerId) {
  const rows = Array.isArray(dashboard?.blocks) ? dashboard.blocks : []
  if (!officerId) return rows

  const selectedOfficer = dashboard?.officers?.find((item) => (
    String(item.officer_id) === String(officerId)
  ))
  if (!selectedOfficer) return []

  return rows.map((item) => ({
    ...item,
    final: selectedOfficer.blocks?.[item.block] ?? 0,
  }))
}

export function blockScoreDomain([dataMin, dataMax]) {
  const minimum = Number.isFinite(dataMin) ? Math.min(0, dataMin) : 0
  const maximum = Number.isFinite(dataMax) ? Math.max(0, dataMax) : 0
  const span = maximum - minimum

  if (span === 0) return [-1, 1]

  const padding = Math.max(span * 0.25, 1)
  return [
    minimum < 0 ? minimum - padding : 0,
    maximum > 0 ? maximum + padding : 0,
  ]
}

export function blockScoreLabelPosition({ x, width, value }) {
  const isNegative = Number(value) < 0

  return isNegative
    ? { x: x - LABEL_EDGE_PADDING, textAnchor: 'end' }
    : { x: x + width + LABEL_EDGE_PADDING, textAnchor: 'start' }
}
