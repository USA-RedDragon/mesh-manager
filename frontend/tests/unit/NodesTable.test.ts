import { describe, it, expect } from 'vitest'
import { h } from 'vue'

describe('NodesTable ETX formatting', () => {
  // Simulate the ETX cell rendering logic
  const renderETXCell = (value: number | null | undefined) => {
    if (value === null || value === undefined) {
      return h('span', {}, '—')
    }
    return h('span', {}, String(value))
  }

  it('should display raw ETX value without division', () => {
    const result = renderETXCell(512)
    expect(result.children).toBe('512')
  })

  it('should display raw ETX value for small numbers', () => {
    const result = renderETXCell(256)
    expect(result.children).toBe('256')
  })

  it('should display raw ETX value for large numbers', () => {
    const result = renderETXCell(1024)
    expect(result.children).toBe('1024')
  })

  it('should display em dash for null value', () => {
    const result = renderETXCell(null)
    expect(result.children).toBe('—')
  })

  it('should display em dash for undefined value', () => {
    const result = renderETXCell(undefined)
    expect(result.children).toBe('—')
  })

  it('should display zero as raw value', () => {
    const result = renderETXCell(0)
    expect(result.children).toBe('0')
  })
})
