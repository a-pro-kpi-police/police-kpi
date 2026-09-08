const IMAGE_TYPES = new Set(['image/jpeg', 'image/png', 'image/webp'])
const IMAGE_TYPES_BY_EXTENSION = {
  '.jpeg': 'image/jpeg',
  '.jpg': 'image/jpeg',
  '.png': 'image/png',
  '.webp': 'image/webp',
}

const VIDEO_TYPES = new Set(['video/mp4', 'video/ogg', 'video/webm'])
const VIDEO_TYPES_BY_EXTENSION = {
  '.mp4': 'video/mp4',
  '.ogv': 'video/ogg',
  '.webm': 'video/webm',
}

const attachmentValue = (attachment, key) => (
  attachment?.[key] ?? attachment?.[`attachment_${key}`]
)

const recognizedType = (attachment, types, extensions) => {
  const contentType = attachmentValue(attachment, 'content_type')?.toLowerCase() || ''
  if (types.has(contentType)) return contentType

  const name = attachmentValue(attachment, 'name')?.toLowerCase() || ''
  const extension = Object.keys(extensions).find((value) => name.endsWith(value))
  return extension ? extensions[extension] : ''
}

export const imageAttachmentType = (attachment) => {
	return recognizedType(attachment, IMAGE_TYPES, IMAGE_TYPES_BY_EXTENSION)
}

export const isImageAttachment = (attachment) => Boolean(imageAttachmentType(attachment))

export const videoAttachmentType = (attachment) => (
  recognizedType(attachment, VIDEO_TYPES, VIDEO_TYPES_BY_EXTENSION)
)

export const isVideoAttachment = (attachment) => Boolean(videoAttachmentType(attachment))

export const entryAttachments = (entry) => {
  if (entry?.attachments?.length) return entry.attachments
  if (!entry?.attachment_name) return []
  return [{
    id: null,
    name: entry.attachment_name,
    content_type: entry.attachment_content_type,
    size: entry.attachment_size,
  }]
}

export const attachmentPath = (entryId, attachment) => (
  attachment.id
    ? `/entries/${entryId}/attachments/${attachment.id}`
    : `/entries/${entryId}/attachment`
)

export const attachmentBlob = (data, contentType) => (
  data.type === contentType ? data : new Blob([data], { type: contentType })
)
