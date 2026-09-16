// Targets and norms describe one month; an empty period means all months.
export function loadMonthlyTargetOverview(api, endpoint, params) {
  return params.month ? api.get(endpoint, { params }) : Promise.resolve({ data: null })
}

export function selectedMonthOverview(overview, month) {
  return month && overview?.month === month ? overview : null
}
