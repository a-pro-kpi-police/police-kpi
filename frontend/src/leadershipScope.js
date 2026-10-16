export const leadershipScopeForm = (employee) => ({
  mode: employee?.leadership_scope?.mode === 'all' ? 'all' : 'selected',
  district_ids: [...(employee?.leadership_scope?.district_ids || [])],
})

export const validLeadershipScope = (role, scope) => role !== 'leadership'
  || scope.mode === 'all'
  || (scope.mode === 'selected' && scope.district_ids.length > 0)

export const leadershipScopePayload = (role, scope) => role === 'leadership'
  ? { mode: scope.mode, district_ids: scope.mode === 'all' ? [] : scope.district_ids.map(Number) }
  : undefined
