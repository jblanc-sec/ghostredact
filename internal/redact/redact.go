package redact

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

type Config struct {
	Mode   string // mask|hash|tag
	Salt   string
	Types  string // explicit comma list (overrides defaults) or blank
	Locale string // comma list of packs, e.g. "br"
	Custom map[string]string // name -> regex (user-supplied)
}

type Redactor struct {
	cfg      Config
	patterns map[string]*regexp.Regexp
	order    []string
	counts   map[string]int
}

func NewRedactor(cfg Config) (*Redactor, error) {
	r := &Redactor{cfg: cfg}

	// Registry of ALL known detectors
	all := map[string]*regexp.Regexp{
		"email": regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`),
		"phone": regexp.MustCompile(`\b(?:\+?\d{1,3}[\s\-.]?)?(?:\(?\d{2,4}\)?[\s\-.]?)?\d{3,5}[\s\-.]?\d{4}\b`),
		"cc":    regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`),
		"ipv4":  regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4]\d|1?\d?\d)\.){3}(?:25[0-5]|2[0-4]\d|1?\d?\d)\b`),
		"ipv6":  regexp.MustCompile(`\b(?:[A-F0-9]{1,4}:){7}[A-F0-9]{1,4}\b`),
		"iban":  regexp.MustCompile(`\b[A-Z]{2}\d{2}[A-Z0-9]{11,30}\b`),

		// Brazil (optional via -locale br)
		"cpf":  regexp.MustCompile(`\b\d{3}\.\d{3}\.\d{3}-\d{2}\b`),
		"cnpj": regexp.MustCompile(`\b\d{2}\.\d{3}\.\d{3}/\d{4}-\d{2}\b`),
		"cep":  regexp.MustCompile(`\b\d{5}-?\d{3}\b`),
		"rg":   regexp.MustCompile(`\b\d{1,2}\.\d{3}\.\d{3}-\d{1}\b`),
	}

	// International default set
	defaultSet := []string{"email", "cc", "phone", "ipv4", "ipv6", "iban"}

	// Locale packs
	packs := map[string][]string{
		"br": {"cpf", "cnpj", "cep", "rg"},
	}

	enabled := map[string]bool{}

	// 1) Start with defaults unless Types explicitly provided
	if strings.TrimSpace(cfg.Types) == "" {
		for _, k := range defaultSet {
			enabled[k] = true
		}
	} else {
		for _, t := range strings.Split(cfg.Types, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				enabled[t] = true
			}
		}
	}

	// 2) Merge locale packs (if any)
	for _, p := range strings.Split(cfg.Locale, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if names, ok := packs[p]; ok {
			for _, n := range names {
				enabled[n] = true
			}
		}
	}

	// Build active pattern map from enabled built-ins
	r.patterns = map[string]*regexp.Regexp{}
	for name := range enabled {
		if re, ok := all[name]; ok {
			r.patterns[name] = re
		}
	}

	// 3) Add user-supplied custom patterns (always enabled)
	if cfg.Custom != nil {
		for name, expr := range cfg.Custom {
			name = strings.TrimSpace(name)
			expr = strings.TrimSpace(expr)
			if name == "" || expr == "" {
				continue
			}
			re, err := regexp.Compile(expr)
			if err != nil {
				return nil, fmt.Errorf("invalid custom regex for %q: %v", name, err)
			}
			r.patterns[name] = re
			// Append to order at end so customs run after built-ins
			r.order = append(r.order, name)
		}
	}

	// Redaction order (stable for known kinds; customs appended after)
	baseOrder := []string{"email", "cc", "phone", "cpf", "cnpj", "rg", "cep", "ipv4", "ipv6", "iban"}
	for _, k := range baseOrder {
		if _, ok := r.patterns[k]; ok {
			r.order = append(r.order, k)
		}
	}
	// ensure customs stay after base by reordering: remove dups while preserving first occurrence
	seen := map[string]bool{}
	var ordered []string
	for _, k := range r.order {
		if !seen[k] {
			seen[k] = true
			ordered = append(ordered, k)
		}
	}
	r.order = ordered

	r.counts = make(map[string]int)
	return r, nil
}

func (r *Redactor) RedactString(s string) string {
	for _, name := range r.order {
		re, ok := r.patterns[name]
		if !ok {
			continue
		}
		switch name {
		case "cc":
			s = re.ReplaceAllStringFunc(s, func(m string) string {
				digits := onlyDigits(m)
				if len(digits) < 13 || len(digits) > 19 || !luhnOK(digits) {
					return m
				}
				r.counts[name]++
				return r.replace(name, m)
			})
		default:
			s = re.ReplaceAllStringFunc(s, func(m string) string {
				r.counts[name]++
				return r.replace(name, m)
			})
		}
	}
	return s
}

func (r *Redactor) replace(kind, match string) string {
	switch r.cfg.Mode {
	case "hash":
		h := sha256.Sum256([]byte(r.cfg.Salt + match))
		return fmt.Sprintf("<%s:%s>", strings.ToUpper(kind), hex.EncodeToString(h[:8]))
	case "tag":
		return fmt.Sprintf("<%s>", strings.ToUpper(kind))
	default: // mask
		switch kind {
		case "email":
			at := strings.Index(match, "@")
			if at <= 1 {
				return "<EMAIL>"
			}
			return match[:1] + strings.Repeat("*", at-1) + match[at:]
		case "cc":
			d := onlyDigits(match)
			if len(d) < 4 {
			 return "<CC>"
			}
			return strings.Repeat("*", len(d)-4) + d[len(d)-4:]
		case "phone":
			return maskKeepLastDigits(match, 4, '*')
		default:
			// For customs and any other kinds, use tag-like generic replacement.
			return "<" + strings.ToUpper(kind) + ">"
		}
	}
}

func (r *Redactor) Counts() map[string]int {
	out := map[string]int{}
	for k, v := range r.counts {
		out[k] = v
	}
	return out
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func luhnOK(s string) bool {
	sum := 0
	alt := false
	for i := len(s) - 1; i >= 0; i-- {
		n := int(s[i] - '0')
		if alt {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alt = !alt
	}
	return sum%10 == 0
}

func maskKeepLastDigits(s string, keep int, ch rune) string {
	seen := 0
	runes := []rune(s)
	for i := len(runes) - 1; i >= 0; i-- {
		r := runes[i]
		if r >= '0' && r <= '9' {
			if seen < keep {
				seen++
			} else {
				runes[i] = ch
			}
		}
	}
	return string(runes)
}
