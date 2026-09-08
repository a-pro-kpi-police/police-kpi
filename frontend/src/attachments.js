const IMAGE_TYPES = new Set(['image/jpeg', 'image/png', 'image/webp'])
const IMAGE_TYPES_BY_EXTENSION = {
  '.jpeg': 'image/jpeg',
  '.jpg': 'image/jpeg',
  '.png': 'image/png',
  '.webp': 'image/webp',
}

export const imageAttachmentType = (attachment) => {
  const contentType = attachment?.attachment_content_type?.toLowerCase() || ''
  if (IMAGE_TYPES.has(contentType)) return contentType

  const name = attachment?.attachment_name?.toLowerCase() || ''
  const extension = Object.keys(IMAGE_TYPES_BY_EXTENSION).find((value) => name.endsWith(value))
  return extension ? IMAGE_TYPES_BY_EXTENSION[extension] : ''
}

export const isImageAttachment = (attachment) => Boolean(imageAttachmentType(attachment))

export const attachmentBlob = (data, contentType) => (
  data.type === contentType ? data : new Blob([data], { type: contentType })
)
