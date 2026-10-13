// Pure validation functions — unit-testable, no framework dependencies

export function isEmail(value: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value.trim())
}

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase()
}

export function isDomain(value: string): boolean {
  if (!value) return true
  const v = value.trim().replace(/^https?:\/\//, '').replace(/\/$/, '')
  return /^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)+$/.test(v)
}

export interface PasswordRules {
  minLength: boolean
  hasUpper: boolean
  hasLower: boolean
  hasDigit: boolean
  hasSymbol: boolean
}

export function checkPasswordRules(value: string): PasswordRules {
  return {
    minLength: value.length >= 12,
    hasUpper: /[A-Z]/.test(value),
    hasLower: /[a-z]/.test(value),
    hasDigit: /\d/.test(value),
    hasSymbol: /[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?`~]/.test(value),
  }
}

export function isPasswordStrong(value: string): boolean {
  const r = checkPasswordRules(value)
  return r.minLength && r.hasUpper && r.hasLower && r.hasDigit && r.hasSymbol
}

export function passwordStrengthScore(value: string): number {
  const r = checkPasswordRules(value)
  let score = 0
  if (r.minLength) score++
  if (r.hasUpper) score++
  if (r.hasLower) score++
  if (r.hasDigit) score++
  if (r.hasSymbol) score++
  if (value.length >= 16) score++
  return Math.min(score, 5)
}

export function passwordsMatch(a: string, b: string): boolean {
  return a === b && a.length > 0
}

export function isRequired(value: string): boolean {
  return value.trim().length > 0
}

export function hasLength(value: string, min: number, max: number): boolean {
  const len = value.trim().length
  return len >= min && len <= max
}

export function parseEmailList(raw: string): { valid: string[]; invalid: string[] } {
  const parts = raw.split(/[\s,\n;]+/).map(s => s.trim()).filter(s => s.length > 0)
  const valid: string[] = []
  const invalid: string[] = []
  const seen = new Set<string>()
  for (const part of parts) {
    const normalized = normalizeEmail(part)
    if (seen.has(normalized)) {
      if (!invalid.includes(`${part} (doublon)`)) invalid.push(`${part} (doublon)`)
      continue
    }
    seen.add(normalized)
    if (isEmail(part)) {
      valid.push(normalized)
    } else {
      invalid.push(part)
    }
  }
  return { valid, invalid }
}

export const COUNTRIES = [
  'France', 'Allemagne', 'Royaume-Uni', 'Espagne', 'Italie', 'Pays-Bas', 'Belgique',
  'Suisse', 'Luxembourg', 'Suède', 'Irlande', 'Portugal', 'Autriche', 'Danemark',
  'Finlande', 'Norvège', 'Pologne', 'États-Unis', 'Canada', 'Brésil', 'Mexique',
  'Japon', 'Singapour', 'Australie', 'Nouvelle-Zélande', 'Maroc', 'Tunisie',
  'Afrique du Sud', 'Inde', 'Autre',
] as const

export const INDUSTRIES = [
  'Technology', 'Finance', 'Healthcare', 'Education', 'Retail', 'Public sector', 'Other',
] as const

export const COMPANY_SIZES = [
  '1-10', '11-50', '51-200', '201-1000', '1000+',
] as const
