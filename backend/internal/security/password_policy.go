package security

import (
	myerrors "backend/internal/errors"
	"backend/model"
	"crypto/rand"
	"math/big"
	"strings"
	"unicode"
)

type passwordCharClass struct {
	name  string
	chars string
}

func ValidatePasswordByConfigure(password string, cfg model.SysConfigure) error {
	pwd := strings.TrimSpace(password)
	if len(pwd) == 0 {
		return myerrors.ErrPasswordEmpty
	}
	for _, r := range pwd {
		if unicode.IsSpace(r) {
			return myerrors.ErrPasswordInvalid
		}
	}
	minLen := cfg.PasswordLength
	if minLen <= 0 {
		minLen = 6
	}
	policyMinLen, policyComplexity := passwordPolicyRequirements(cfg.PasswordPolicy)
	if minLen < policyMinLen {
		minLen = policyMinLen
	}
	if len(pwd) < minLen {
		return myerrors.ErrPasswordTooShort
	}

	hasLower, hasUpper, hasDigit, hasSpecial := false, false, false, false
	for _, r := range pwd {
		switch {
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= '0' && r <= '9':
			hasDigit = true
		default:
			// 允许所有非空白字符作为特殊字符
			hasSpecial = true
		}
	}

	complexity := cfg.PasswordComplexity
	if complexity <= 0 {
		complexity = 1
	}
	if complexity < policyComplexity {
		complexity = policyComplexity
	}

	// 复杂度定义（低/中/高）：
	// 1 低：仅要求长度
	// 2 中：至少包含「字母 + 数字」
	// 3 高：至少包含 3 类字符（大写/小写/数字/特殊字符），且必须包含字母和数字
	if complexity >= 2 {
		if !(hasDigit && (hasLower || hasUpper)) {
			return myerrors.ErrPasswordTooSimple
		}
	}
	if complexity >= 3 {
		classCount := 0
		if hasLower {
			classCount++
		}
		if hasUpper {
			classCount++
		}
		if hasDigit {
			classCount++
		}
		if hasSpecial {
			classCount++
		}
		if classCount < 3 {
			return myerrors.ErrPasswordNotComplexEnough
		}
	}

	return nil
}

func GeneratePasswordByConfigure(cfg model.SysConfigure) (string, error) {
	minLen := cfg.PasswordLength
	if minLen <= 0 {
		minLen = 6
	}
	complexity := cfg.PasswordComplexity
	if complexity <= 0 {
		complexity = 1
	}
	policyMinLen, policyComplexity := passwordPolicyRequirements(cfg.PasswordPolicy)
	if minLen < policyMinLen {
		minLen = policyMinLen
	}
	if complexity < policyComplexity {
		complexity = policyComplexity
	}

	lower := passwordCharClass{name: "lower", chars: "abcdefghijklmnopqrstuvwxyz"}
	upper := passwordCharClass{name: "upper", chars: "ABCDEFGHIJKLMNOPQRSTUVWXYZ"}
	digit := passwordCharClass{name: "digit", chars: "0123456789"}
	special := passwordCharClass{name: "special", chars: "!@#$%^&*_-+=?"}

	var required []passwordCharClass
	switch {
	case complexity >= 3:
		required = []passwordCharClass{lower, upper, digit}
		// 高复杂度下额外加入特殊字符，提升强度
		required = append(required, special)
	case complexity == 2:
		required = []passwordCharClass{lower, digit}
	default:
		required = []passwordCharClass{lower, digit}
	}

	length := minLen
	if length < len(required) {
		length = len(required)
	}

	allChars := lower.chars + upper.chars + digit.chars + special.chars
	buf := make([]byte, 0, length)

	// 先放入每个必选类 1 个字符
	for _, cls := range required {
		ch, err := pickOne(cls.chars)
		if err != nil {
			return "", err
		}
		buf = append(buf, ch)
	}
	// 填充剩余
	for len(buf) < length {
		ch, err := pickOne(allChars)
		if err != nil {
			return "", err
		}
		buf = append(buf, ch)
	}
	// 洗牌
	if err := shuffleBytes(buf); err != nil {
		return "", err
	}

	pwd := string(buf)
	if err := ValidatePasswordByConfigure(pwd, cfg); err != nil {
		// 随机结果在极端长度配置下仍可能不满足组合规则，此时重新生成。
		return GeneratePasswordByConfigure(cfg)
	}
	return pwd, nil
}

func passwordPolicyRequirements(policy string) (int, int) {
	switch strings.ToLower(strings.TrimSpace(policy)) {
	case "low":
		return 6, 1
	case "high":
		return 12, 3
	case "strong":
		return 12, 3
	case "medium":
		return 8, 2
	case "custom":
		return 0, 0
	default:
		return 0, 0
	}
}

func pickOne(chars string) (byte, error) {
	max := big.NewInt(int64(len(chars)))
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}
	return chars[n.Int64()], nil
}

func shuffleBytes(b []byte) error {
	for i := len(b) - 1; i > 0; i-- {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		j := int(n.Int64())
		b[i], b[j] = b[j], b[i]
	}
	return nil
}
