const LABEL_EDGE_PADDING = 6
const APPROXIMATE_CHARACTER_WIDTH = 6.5

export function formatBlockScore(value, pointsLabel) {
  return `${value ?? 0} ${pointsLabel}`
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

export function blockScoreLabelPosition({ x, width, value, label }) {
  const isNegative = Number(value) < 0
  const estimatedLabelWidth = label.length * APPROXIMATE_CHARACTER_WIDTH
  const fitsInside = width >= estimatedLabelWidth + LABEL_EDGE_PADDING * 2

  if (fitsInside) {
    return isNegative
      ? { x: x + LABEL_EDGE_PADDING, textAnchor: 'start', inside: true }
      : { x: x + width - LABEL_EDGE_PADDING, textAnchor: 'end', inside: true }
  }

  return isNegative
    ? { x: x - LABEL_EDGE_PADDING, textAnchor: 'end', inside: false }
    : { x: x + width + LABEL_EDGE_PADDING, textAnchor: 'start', inside: false }
}
