export function formatPasscode(code: string): string {
  const digits = code.replace(/\D/g, '').slice(0, 6)
  if (digits.length <= 3) return digits
  return `${digits.slice(0, 3)} ${digits.slice(3)}`
}

export function onlyDigits(value: string, max = 6): string {
  return value.replace(/\D/g, '').slice(0, max)
}

export function relativeTime(_iso: string, label: string): string {
  return label
}

export function uid(prefix = 'id'): string {
  return `${prefix}-${Math.random().toString(36).slice(2, 10)}`
}
