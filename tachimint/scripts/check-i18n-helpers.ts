export function flattenStringValues(value: unknown, prefix = ''): Record<string, string> {
  if (typeof value === 'string') {
    return { [prefix]: value }
  }

  if (!value || typeof value !== 'object') {
    return {}
  }

  if (Array.isArray(value)) {
    return value.reduce<Record<string, string>>((acc, child, index) => {
      const nextPrefix = prefix ? `${prefix}.${index}` : `${index}`
      return { ...acc, ...flattenStringValues(child, nextPrefix) }
    }, {})
  }

  return Object.entries(value).reduce<Record<string, string>>((acc, [key, child]) => {
    const nextPrefix = prefix ? `${prefix}.${key}` : key
    return { ...acc, ...flattenStringValues(child, nextPrefix) }
  }, {})
}

export function extractInterpolationTokens(value: string): string[] {
  const tokens = [...value.matchAll(/\{\{\s*([a-zA-Z0-9_]+)\s*\}\}/g)].map(([, token]) => token)
  return [...new Set(tokens)].sort()
}
