export function parseCpcAmount(value: string): number | null {
  const normalizedValue = value.trim()
  if (!/^(?:\d+|\d*\.\d+)$/.test(normalizedValue)) {
    return null
  }

  const amount = Number(normalizedValue)
  return Number.isFinite(amount) ? amount : null
}
