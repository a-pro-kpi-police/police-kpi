import assert from 'node:assert/strict'
import test from 'node:test'
import { attachmentBlob, imageAttachmentType, isImageAttachment } from './attachments.js'

test('recognizes image attachments by MIME type or supported extension', () => {
  assert.equal(isImageAttachment({ attachment_content_type: 'image/png', attachment_name: 'scan.bin' }), true)
  assert.equal(isImageAttachment({ attachment_content_type: '', attachment_name: 'evidence.JPEG' }), true)
  assert.equal(isImageAttachment({ attachment_content_type: 'application/pdf', attachment_name: 'report.pdf' }), false)
  assert.equal(isImageAttachment({ attachment_content_type: 'image/svg+xml', attachment_name: 'vector.svg' }), false)
  assert.equal(imageAttachmentType({ attachment_content_type: '', attachment_name: 'photo.webp' }), 'image/webp')
})

test('restores the stored image MIME type on downloaded blobs', () => {
  const source = new Blob(['image bytes'], { type: 'application/octet-stream' })
  const result = attachmentBlob(source, 'image/webp')

  assert.equal(result.type, 'image/webp')
  assert.equal(result.size, source.size)
})
