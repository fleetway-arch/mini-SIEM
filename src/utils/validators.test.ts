import { describe, it, expect } from 'vitest'
import {
  isEmail, normalizeEmail, isDomain, checkPasswordRules, isPasswordStrong,
  passwordStrengthScore, passwordsMatch, isRequired, hasLength, parseEmailList,
} from './validators'

describe('isEmail', () => {
  it.each([
    ['alice@example.com', true],
    ['  bob@corp.io  ', true],
    ['no-at-sign.com', false],
    ['two@@example.com', false],
    ['spaces in@example.com', false],
    ['missing@tld', false],
    ['', false],
  ])('%s -> %s', (value, expected) => {
    expect(isEmail(value)).toBe(expected)
  })
})

describe('normalizeEmail', () => {
  it('trims and lower-cases', () => {
    expect(normalizeEmail('  Alice@Example.COM ')).toBe('alice@example.com')
  })
})

describe('isDomain', () => {
  it('accepts empty (optional field), bare domains and URLs', () => {
    expect(isDomain('')).toBe(true)
    expect(isDomain('example.com')).toBe(true)
    expect(isDomain('https://sub.example.co.uk/')).toBe(true)
  })
  it('rejects invalid values', () => {
    expect(isDomain('not a domain')).toBe(false)
    expect(isDomain('-bad.com')).toBe(false)
    expect(isDomain('localhost')).toBe(false)
  })
})

describe('password rules (same as the server)', () => {
  it('requires 12+ characters with upper, lower, digit and symbol', () => {
    expect(isPasswordStrong('Str0ng!Passw0rd')).toBe(true)
    expect(isPasswordStrong('Short1!a')).toBe(false)
    expect(isPasswordStrong('alllowercase1!')).toBe(false)
    expect(isPasswordStrong('ALLUPPERCASE1!')).toBe(false)
    expect(isPasswordStrong('NoDigitsHere!!')).toBe(false)
    expect(isPasswordStrong('NoSymbols12345')).toBe(false)
  })

  it('reports each rule separately', () => {
    expect(checkPasswordRules('abc')).toEqual({
      minLength: false, hasUpper: false, hasLower: true, hasDigit: false, hasSymbol: false,
    })
  })

  it('scores strength from 0 to 5', () => {
    expect(passwordStrengthScore('')).toBe(0)
    expect(passwordStrengthScore('Str0ng!Passw0rd')).toBe(5)
    expect(passwordStrengthScore('Str0ng!Passw0rd-and-long')).toBe(5)
  })

  it('checks confirmation', () => {
    expect(passwordsMatch('a', 'a')).toBe(true)
    expect(passwordsMatch('a', 'b')).toBe(false)
    expect(passwordsMatch('', '')).toBe(false)
  })
})

describe('generic validators', () => {
  it('isRequired ignores whitespace', () => {
    expect(isRequired('  ')).toBe(false)
    expect(isRequired(' x ')).toBe(true)
  })
  it('hasLength uses trimmed length', () => {
    expect(hasLength('  ab  ', 2, 3)).toBe(true)
    expect(hasLength('abcd', 2, 3)).toBe(false)
  })
})

describe('parseEmailList', () => {
  it('splits on commas, spaces, semicolons and new lines, normalizes and flags duplicates', () => {
    const result = parseEmailList('Alice@corp.io, bob@corp.io\ncarol@corp.io; alice@CORP.io nope')
    expect(result.valid).toEqual(['alice@corp.io', 'bob@corp.io', 'carol@corp.io'])
    expect(result.invalid).toEqual(['alice@CORP.io (doublon)', 'nope'])
  })
})
