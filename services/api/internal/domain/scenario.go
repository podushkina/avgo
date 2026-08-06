package domain

import "fmt"

type Role string

const (
	RoleBuyer  Role = "buyer"
	RoleSeller Role = "seller"
)

func ParseRole(s string) (Role, error) {
	switch Role(s) {
	case RoleBuyer:
		return RoleBuyer, nil
	case RoleSeller:
		return RoleSeller, nil
	default:
		return "", fmt.Errorf("неизвестная роль %q: ожидается buyer или seller", s)
	}
}

type Verdict string

const (
	VerdictSafe      Verdict = "safe"
	VerdictRisky     Verdict = "risky"
	VerdictDangerous Verdict = "dangerous"
)

func (v Verdict) Points() int {
	switch v {
	case VerdictSafe:
		return 10
	case VerdictRisky:
		return 4
	default:
		return 0
	}
}

type Option struct {
	Text    string  `json:"text"`
	Verdict Verdict `json:"verdict"`
	Outcome string  `json:"outcome"`
}

type Scenario struct {
	ID            int
	Role          Role
	OrderIndex    int
	Title         string
	Situation     string
	Question      string
	Options       []Option
	CorrectOption int
	Explanation   string
	RedFlags      []string
}

type PublicScenario struct {
	ID         int      `json:"id"`
	Role       Role     `json:"role"`
	OrderIndex int      `json:"order_index"`
	Title      string   `json:"title"`
	Situation  string   `json:"situation"`
	Question   string   `json:"question"`
	Options    []string `json:"options"`
}

func (s Scenario) Public() PublicScenario {
	texts := make([]string, 0, len(s.Options))
	for _, o := range s.Options {
		texts = append(texts, o.Text)
	}
	return PublicScenario{
		ID:         s.ID,
		Role:       s.Role,
		OrderIndex: s.OrderIndex,
		Title:      s.Title,
		Situation:  s.Situation,
		Question:   s.Question,
		Options:    texts,
	}
}

func (s Scenario) Option(i int) (Option, bool) {
	if i < 0 || i >= len(s.Options) {
		return Option{}, false
	}
	return s.Options[i], true
}

func (s Scenario) OptionText(i int) string {
	o, ok := s.Option(i)
	if !ok {
		return ""
	}
	return o.Text
}

func (s Scenario) MaxPoints() int {
	best := 0
	for _, o := range s.Options {
		if p := o.Verdict.Points(); p > best {
			best = p
		}
	}
	return best
}
