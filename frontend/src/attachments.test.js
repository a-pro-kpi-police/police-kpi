import assert from 'node:assert/strict'
import test from 'node:test'
import {
  attachmentBlob, attachmentPath, entryAttachments, imageAttachmentType,
  isImageAttachment, isVideoAttachment, videoAttachmentType,
} from './attachments.js'

test('recognizes image attachments by MIME type or supported extension', () => {
  assert.equal(isImageAttachment({ attachment_content_type: 'image/png', attachment_name: 'scan.bin' }), true)
  assert.equal(isImageAttachment({ attachment_content_type: '', attachment_name: 'evidence.JPEG' }), true)
  assert.equal(isImageAttachment({ attachment_content_type: 'application/pdf', attachment_name: 'report.pdf' }), false)
  assert.equal(isImageAttachment({ attachment_content_type: 'image/svg+xml', attachment_name: 'vector.svg' }), false)
  assert.equal(imageAttachmentType({ attachment_content_type: '', attachment_name: 'photo.webp' }), 'image/webp')
})

test('recognizes supported video attachments by MIME type or extension', () => {
  assert.equal(isVideoAttachment({ content_type: 'video/mp4', name: 'clip.bin' }), true)
  assert.equal(videoAttachmentType({ content_type: '', name: 'evidence.WEBM' }), 'video/webm')
  assert.equal(isVideoAttachment({ content_type: 'video/quicktime', name: 'clip.mov' }), false)
  assert.equal(isVideoAttachment({ content_type: 'application/pdf', name: 'clip.pdf' }), false)
})

test('normalizes current and legacy entry attachments', () => {
  const current = [{ id: 9, name: 'clip.mp4', content_type: 'video/mp4', size: 12 }]
  assert.deepEqual(entryAttachments({ attachments: current }), current)
  assert.deepEqual(entryAttachments({
    attachment_name: 'photo.jpg', attachment_content_type: 'image/jpeg', attachment_size: 7,
  }), [{ id: null, name: 'photo.jpg', content_type: 'image/jpeg', size: 7 }])
  assert.equal(attachmentPath(4, current[0]), '/entries/4/attachments/9')
  assert.equal(attachmentPath(4, { id: null }), '/entries/4/attachment')
})

test('restores the stored image MIME type on downloaded blobs', () => {
  const source = new Blob(['image bytes'], { type: 'application/octet-stream' })
  const result = attachmentBlob(source, 'image/webp')

  assert.equal(result.type, 'image/webp')
  assert.equal(result.size, source.size)
})
