const timestampWithZonePattern = /(?:Z|[+-]\d{2}:\d{2})$/

export const parseApiTimestamp = (value) => {
  const timestamp = String(value || '')
  return new Date(timestampWithZonePattern.test(timestamp) ? timestamp : `${timestamp}Z`)
}

const messageTime = (message) => {
  const time = parseApiTimestamp(message.created_at).getTime()
  return Number.isFinite(time) ? time : 0
}

const messageId = (message) => {
  const id = Number(message.id)
  return Number.isFinite(id) ? id : 0
}

export const mergeSupportMessages = (current, incoming) => {
  const byId = new Map()

  ;[...(Array.isArray(current) ? current : []), ...(Array.isArray(incoming) ? incoming : [])]
    .filter((message) => message && message.id != null)
    .forEach((message) => {
      byId.set(String(message.id), message)
    })

  return Array.from(byId.values()).sort((left, right) => {
    const timeDiff = messageTime(left) - messageTime(right)
    if (timeDiff !== 0) return timeDiff
    return messageId(left) - messageId(right)
  })
}
