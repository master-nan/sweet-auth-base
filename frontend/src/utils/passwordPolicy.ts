import { translate as t } from '@/boot/i18n'
import type { Configure } from '@/api/services/basic'

export type PasswordRule = (val: unknown) => true | string

function hasWhitespace(s: string): boolean {
  return /\s/.test(s)
}

function hasLower(s: string): boolean {
  return /[a-z]/.test(s)
}

function hasUpper(s: string): boolean {
  return /[A-Z]/.test(s)
}

function hasDigit(s: string): boolean {
  return /\d/.test(s)
}

function hasSpecial(s: string): boolean {
  // 与后端保持一致：只要是非字母非数字（且非空白）都视为特殊字符
  return /[^A-Za-z0-9\s]/.test(s)
}

export type PasswordPolicyConfig = Pick<Configure, 'password_length' | 'password_complexity'> &
  Partial<Pick<Configure, 'password_policy'>>

export type PasswordPolicyLevel = 'low' | 'medium' | 'strong' | 'high' | 'custom'

export interface PasswordPolicyPreset {
  value: PasswordPolicyLevel
  label: string
  shortLabel: string
  minLen: number
  complexity: number
  description: string
  regexText: string
}

export const passwordPolicyPresets: PasswordPolicyPreset[] = [
  {
    value: 'low',
    get label() {
      return t('ui.lowAtLeast6Places')
    },
    get shortLabel() {
      return t('ui.low')
    },
    minLen: 6,
    complexity: 1,
    get description() {
      return t('ui.onlyVerifyTheLengthWithoutLimitingTheCombinationOfCharactersSuitableFor')
    },
    regexText: '^\\S{6,}$',
  },
  {
    value: 'medium',
    get label() {
      return t('ui.mediumAtLeast8BitsLettersNumbers')
    },
    get shortLabel() {
      return t('ui.medium')
    },
    minLen: 8,
    complexity: 2,
    get description() {
      return t('ui.atLeastIncludeLettersAndNumbersBlankCharactersAreNotAllowedAnd')
    },
    regexText: '^(?=.*[A-Za-z])(?=.*\\d)\\S{8,}$',
  },
  {
    value: 'high',
    get label() {
      return t('ui.highAtLeast12BitsThreeCharacters')
    },
    get shortLabel() {
      return t('ui.high')
    },
    minLen: 12,
    complexity: 3,
    get description() {
      return t('ui.containsAtLeastThreeTypesOfCharactersAndMustContainLettersAnd')
    },
    regexText:
      '^(?=.*[A-Za-z])(?=.*\\d)(?:(?=.*[a-z])(?=.*[A-Z])|(?=.*[a-z])(?=.*[^A-Za-z0-9\\s])|(?=.*[A-Z])(?=.*[^A-Za-z0-9\\s])|(?=.*\\d)(?=.*[^A-Za-z0-9\\s]))\\S{12,}$',
  },
  {
    value: 'custom',
    get label() {
      return t('ui.customManualSettingsLengthAndGrouping')
    },
    get shortLabel() {
      return t('ui.custom')
    },
    minLen: 6,
    complexity: 1,
    get description() {
      return t('ui.useTheMinimumLengthAndComplexityConfigurationBelowToFitTheSpecial')
    },
    get regexText() {
      return t('ui.validationByCustomLengthAndComplexityDynamic')
    },
  },
]

export const passwordPolicyOptions = passwordPolicyPresets.map((preset) => ({
  label: preset.label,
  value: preset.value,
  description: preset.description,
}))

export function normalizePasswordPolicy(policy?: string): PasswordPolicyLevel {
  const normalized = String(policy ?? '')
    .trim()
    .toLowerCase()
  if (normalized === 'strong') return 'high'
  if (['low', 'medium', 'high', 'custom'].includes(normalized)) {
    return normalized as PasswordPolicyLevel
  }
  return 'medium'
}

export function getPasswordPolicyPreset(policy?: string): PasswordPolicyPreset {
  const normalized = normalizePasswordPolicy(policy)
  return (
    passwordPolicyPresets.find((preset) => preset.value === normalized) ?? passwordPolicyPresets[1]!
  )
}

export function effectivePasswordPolicy(cfg: PasswordPolicyConfig): {
  minLen: number
  complexity: number
} {
  const normalizedPolicy = normalizePasswordPolicy(cfg.password_policy)
  if (normalizedPolicy === 'custom') {
    return {
      minLen: Math.max(6, Number(cfg.password_length) || 0),
      complexity: Math.max(1, Number(cfg.password_complexity) || 0),
    }
  }
  const preset = getPasswordPolicyPreset(normalizedPolicy)
  return {
    minLen: preset.minLen,
    complexity: preset.complexity,
  }
}

export function passwordPolicyRegexText(cfg: PasswordPolicyConfig): string {
  const normalizedPolicy = normalizePasswordPolicy(cfg.password_policy)
  if (normalizedPolicy !== 'custom') {
    return getPasswordPolicyPreset(normalizedPolicy).regexText
  }
  const { minLen, complexity } = effectivePasswordPolicy(cfg)
  if (complexity <= 1) {
    return `^\\S{${minLen},}$`
  }
  if (complexity === 2) {
    return `^(?=.*[A-Za-z])(?=.*\\d)\\S{${minLen},}$`
  }
  return `^(?=.*[A-Za-z])(?=.*\\d)(?:(?=.*[a-z])(?=.*[A-Z])|(?=.*[a-z])(?=.*[^A-Za-z0-9\\s])|(?=.*[A-Z])(?=.*[^A-Za-z0-9\\s])|(?=.*\\d)(?=.*[^A-Za-z0-9\\s]))\\S{${minLen},}$`
}

export function passwordPolicyDescription(cfg: PasswordPolicyConfig): string {
  const { minLen, complexity } = effectivePasswordPolicy(cfg)

  if (complexity <= 1) {
    return t('ui.atLeastBits', { minLen: minLen })
  }

  if (complexity === 2) {
    return t('ui.atLeastBitsWithLettersAndNumbers', { minLen: minLen })
  }

  // complexity >= 3
  return t('ui.atLeastBitsWhichContainAtLeastThreeTypesOfCharactersFast', { minLen: minLen })
}

export function validatePasswordByConfigure(
  password: string,
  cfg: PasswordPolicyConfig,
): true | string {
  const pwd = String(password ?? '').trim()
  if (!pwd) return t('ui.passwordCannotBeEmpty')
  if (hasWhitespace(pwd)) return t('ui.passwordCannotContainWhitespaceCharacters')

  const { minLen, complexity } = effectivePasswordPolicy(cfg)
  if (pwd.length < minLen) return t('ui.passwordLengthInsufficientAtLeastBit', { minLen: minLen })

  if (complexity <= 1) return true

  const lower = hasLower(pwd)
  const upper = hasUpper(pwd)
  const digit = hasDigit(pwd)
  const special = hasSpecial(pwd)

  if (complexity >= 2) {
    if (!(digit && (lower || upper))) {
      return t('ui.passwordComplexityInsufficientContainsAtLeastLettersAndNumbers')
    }
  }

  if (complexity >= 3) {
    const classCount = [lower, upper, digit, special].filter(Boolean).length
    if (classCount < 3) {
      return t('ui.insufficientPasswordComplexityContainsAtLeastThreeTypesOfCharactersFacileCut')
    }
  }

  return true
}

export function buildPasswordRules(cfg: PasswordPolicyConfig): PasswordRule[] {
  return [
    (val) => {
      const password = typeof val === 'string' ? val : ''
      const result = validatePasswordByConfigure(password, cfg)
      return result === true ? true : result
    },
  ]
}
