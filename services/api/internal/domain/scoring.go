package domain

import "math"

type Answer struct {
	ScenarioID int `json:"scenario_id"`
	Option     int `json:"option"`
}

type Review struct {
	ScenarioID        int      `json:"scenario_id"`
	Title             string   `json:"title"`
	Question          string   `json:"question"`
	Answered          bool     `json:"answered"`
	IsCorrect         bool     `json:"is_correct"`
	YourOption        int      `json:"your_option"`
	YourOptionText    string   `json:"your_option_text"`
	YourVerdict       Verdict  `json:"your_verdict"`
	YourOutcome       string   `json:"your_outcome"`
	CorrectOption     int      `json:"correct_option"`
	CorrectOptionText string   `json:"correct_option_text"`
	CorrectOutcome    string   `json:"correct_outcome"`
	Explanation       string   `json:"explanation"`
	RedFlags          []string `json:"red_flags"`
	Points            int      `json:"points"`
}

type Result struct {
	Correct  int      `json:"correct"`
	Total    int      `json:"total"`
	Percent  float64  `json:"percent"`
	Score    int      `json:"score"`
	MaxScore int      `json:"max_score"`
	Level    string   `json:"level"`
	Perfect  bool     `json:"perfect"`
	Reviews  []Review `json:"reviews"`
	Mistakes []Review `json:"mistakes"`
	RedFlags []string `json:"missed_red_flags"`
}

func Score(scenarios []Scenario, answers []Answer) Result {
	chosen := make(map[int]int, len(answers))
	for _, a := range answers {
		chosen[a.ScenarioID] = a.Option
	}

	res := Result{
		Total:    len(scenarios),
		Reviews:  []Review{},
		Mistakes: []Review{},
		RedFlags: []string{},
	}
	seenFlag := map[string]bool{}

	for _, s := range scenarios {
		res.MaxScore += s.MaxPoints()

		opt, answered := chosen[s.ID]
		if !answered {
			opt = -1
		}
		picked, valid := s.Option(opt)

		rev := Review{
			ScenarioID:        s.ID,
			Title:             s.Title,
			Question:          s.Question,
			Answered:          answered && valid,
			IsCorrect:         answered && opt == s.CorrectOption,
			YourOption:        opt,
			YourOptionText:    picked.Text,
			YourVerdict:       picked.Verdict,
			YourOutcome:       picked.Outcome,
			CorrectOption:     s.CorrectOption,
			CorrectOptionText: s.OptionText(s.CorrectOption),
			Explanation:       s.Explanation,
			RedFlags:          s.RedFlags,
		}
		if correct, ok := s.Option(s.CorrectOption); ok {
			rev.CorrectOutcome = correct.Outcome
		}
		if rev.Answered {
			rev.Points = picked.Verdict.Points()
			res.Score += rev.Points
		}

		if rev.IsCorrect {
			res.Correct++
		} else {
			res.Mistakes = append(res.Mistakes, rev)
			for _, f := range s.RedFlags {
				if !seenFlag[f] {
					seenFlag[f] = true
					res.RedFlags = append(res.RedFlags, f)
				}
			}
		}
		res.Reviews = append(res.Reviews, rev)
	}

	if res.Total > 0 {
		res.Percent = math.Round(float64(res.Correct)/float64(res.Total)*1000) / 10
	}
	res.Perfect = res.Total > 0 && res.Correct == res.Total
	res.Level = LevelFor(res.Percent)

	return res
}

func LevelFor(percent float64) string {
	switch {
	case percent >= 100:
		return "Эксперт по безопасным сделкам"
	case percent >= 80:
		return "Уверенный пользователь"
	case percent >= 50:
		return "Есть пробелы"
	default:
		return "Высокий риск"
	}
}

type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

func SuggestDifficulty(percent float64) Difficulty {
	switch {
	case percent >= 85:
		return DifficultyHard
	case percent >= 50:
		return DifficultyMedium
	default:
		return DifficultyEasy
	}
}
