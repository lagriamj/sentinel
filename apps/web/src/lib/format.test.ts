import { describe, expect, it } from 'vitest'
import { canCancel, canDelete } from '../types/scan-job'
import { formatDateTime, targetEndpoint } from './format'

describe('scan job actions', () => {
  it('matches the API cancel and delete rules', () => {
    expect(canCancel('pending')).toBe(true)
    expect(canCancel('queued')).toBe(true)
    expect(canCancel('running')).toBe(true)
    expect(canCancel('completed')).toBe(false)
    expect(canCancel('failed')).toBe(false)
    expect(canCancel('cancelled')).toBe(false)

    expect(canDelete('pending')).toBe(true)
    expect(canDelete('completed')).toBe(true)
    expect(canDelete('failed')).toBe(true)
    expect(canDelete('cancelled')).toBe(true)
    expect(canDelete('queued')).toBe(false)
    expect(canDelete('running')).toBe(false)
  })
})

describe('formatDateTime', () => {
  it('returns a dash for missing or invalid values', () => {
    expect(formatDateTime(undefined)).toBe('—')
    expect(formatDateTime('not-a-date')).toBe('—')
  })

  it('formats a timestamp', () => {
    expect(formatDateTime('2026-11-27T08:00:00Z')).not.toBe('—')
  })
})

describe('targetEndpoint', () => {
  it('joins host and port', () => {
    expect(targetEndpoint('example.com', 443)).toBe('example.com:443')
  })
})
