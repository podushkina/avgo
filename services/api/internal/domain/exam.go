package domain

import "math"

type Verdict string

const (
	VerdictPassed Verdict = "passed"
	VerdictFailed Verdict = "failed"
)

type Mistake string

const (
	MistakeLeakedSMSCode     Mistake = "leaked_sms_code"
	MistakeLeakedCardData    Mistake = "leaked_card_data"
	MistakeAgreedPayLink     Mistake = "agreed_to_pay_via_link"
	MistakeAgreedOffPlatform Mistake = "agreed_off_platform"
	MistakeSharedPhone       Mistake = "shared_phone"
	MistakeMovedToMessenger  Mistake = "moved_to_messenger"
)

var criticalMistakes = map[Mistake]bool{
	MistakeLeakedSMSCode:     true,
	MistakeLeakedCardData:    true,
	MistakeAgreedPayLink:     true,
	MistakeAgreedOffPlatform: true,
}

func (m Mistake) IsCritical() bool { return criticalMistakes[m] }

var mistakeTitles = map[Mistake]string{
	MistakeLeakedSMSCode:     "Вы сообщили код из СМС",
	MistakeLeakedCardData:    "Вы сообщили данные карты",
	MistakeAgreedPayLink:     "Вы согласились оплатить по присланной ссылке",
	MistakeAgreedOffPlatform: "Вы согласились на оплату вне платформы",
	MistakeSharedPhone:       "Вы дали номер телефона",
	MistakeMovedToMessenger:  "Вы согласились уйти в сторонний мессенджер",
}

func (m Mistake) Title() string {
	if t, ok := mistakeTitles[m]; ok {
		return t
	}
	return string(m)
}

func HasCritical(ms []Mistake) bool {
	for _, m := range ms {
		if m.IsCritical() {
			return true
		}
	}
	return false
}

type EndReason string

const (
	EndCritical    EndReason = "critical_mistake"
	EndRefused     EndReason = "refused_and_ended"
	EndTacticsDone EndReason = "tactics_exhausted"
	EndUserFinish  EndReason = "user_finished"
	EndLimit       EndReason = "limit_reached"
)

type ExamOutcome struct {
	Verdict      Verdict
	Explanation  string
	Mistakes     []Mistake
	Reason       EndReason
	TacticsFaced int
}

const explanationSoft = "Критических ошибок вы не допустили, но пара уступок всё же была. " +
	"Мошеннику этого обычно достаточно, чтобы вернуться к теме в следующем разговоре."

func Decide(mistakes []Mistake, reason EndReason, tacticsFaced int) ExamOutcome {
	if HasCritical(mistakes) {
		return ExamOutcome{
			Verdict:      VerdictFailed,
			Explanation:  criticalExplanation(mistakes),
			Mistakes:     mistakes,
			Reason:       EndCritical,
			TacticsFaced: tacticsFaced,
		}
	}

	out := ExamOutcome{
		Verdict:      VerdictPassed,
		Mistakes:     mistakes,
		Reason:       reason,
		TacticsFaced: tacticsFaced,
	}

	switch {
	case len(mistakes) > 0:
		out.Explanation = explanationSoft
	case reason == EndRefused:
		out.Explanation = "Вы распознали мошенника и вышли из разговора, не отдав ничего ценного. " +
			"В реальной сделке это правильный финал: с таким собеседником говорить не о чем."
	case reason == EndTacticsDone:
		out.Explanation = "Собеседник перепробовал все свои приёмы, и ни один не сработал. " +
			"Вы не назвали код, не отдали данные карты и не согласились платить в обход площадки."
	case reason == EndUserFinish && tacticsFaced == 0:
		out.Explanation = "Вы завершили разговор до того, как собеседник успел что-то предложить. " +
			"Формально вы ничего не потеряли, но и проверить себя толком не успели - попробуйте пройти ещё раз."
	case reason == EndUserFinish:
		out.Explanation = "Вы сами прервали разговор и ничего мошеннику не отдали. " +
			"Прекратить общение при первых признаках обмана - совершенно правильная реакция."
	default:
		out.Explanation = "Вы прошли весь разговор и не отдали мошеннику ничего ценного: " +
			"ни кода из СМС, ни данных карты, ни согласия платить вне платформы."
	}

	return out
}

func criticalExplanation(mistakes []Mistake) string {
	for _, m := range mistakes {
		if !m.IsCritical() {
			continue
		}
		switch m {
		case MistakeLeakedSMSCode:
			return "Вы продиктовали код из СМС. Такой код подтверждает вход в ваш аккаунт " +
				"или списание денег, и настоящему покупателю он не нужен никогда."
		case MistakeLeakedCardData:
			return "Вы сообщили данные карты. Номера, срока и кода с обратной стороны " +
				"достаточно, чтобы оплачивать покупки от вашего имени."
		case MistakeAgreedPayLink:
			return "Вы согласились перейти по присланной ссылке и оплатить там. " +
				"Именно ради этого перехода и затевалась вся переписка."
		case MistakeAgreedOffPlatform:
			return "Вы согласились на оплату вне платформы. Такой перевод не защищён " +
				"сделкой и его невозможно оспорить."
		}
	}
	return "Вы допустили критическую ошибку, которая в реальной сделке стоила бы денег или аккаунта."
}

func ExamScore(o ExamOutcome) float64 {
	switch {
	case HasCritical(o.Mistakes):
		return 0
	case len(o.Mistakes) > 0:
		return 0.5
	case o.Reason == EndUserFinish && o.TacticsFaced == 0:
		return 0.5
	default:
		return 1
	}
}

func Score(correctSteps, totalSteps int, examScore float64) int {
	trainingPart := 0.0
	if totalSteps > 0 {
		trainingPart = float64(correctSteps) / float64(totalSteps)
	}
	return int(math.Round(60*trainingPart + 40*examScore))
}

func Grade(score int) string {
	switch {
	case score >= 90:
		return "Эксперт"
	case score >= 70:
		return "Уверенный"
	case score >= 40:
		return "Осторожный"
	default:
		return "Новичок"
	}
}

var fallbackTips = map[Mistake]string{
	MistakeLeakedSMSCode:     "Код из СМС не сообщается никому и ни в какой части - ни продавцу, ни «поддержке».",
	MistakeLeakedCardData:    "Для зачисления денег достаточно номера карты. Срок и код с обратной стороны нужны только для списания.",
	MistakeAgreedPayLink:     "Статус заказа и оплату проверяйте внутри приложения, а не по ссылке из переписки.",
	MistakeAgreedOffPlatform: "Платите при получении или через сделку на площадке - перевод вне её не возвращается.",
	MistakeSharedPhone:       "Номер телефона не нужен для сделки: по нему подбирают пароли и рассылают фишинг.",
	MistakeMovedToMessenger:  "Оставайтесь в чате площадки: вне его не работают ни защита сделки, ни поддержка.",
}

const (
	tipDefaultKeepLine = "Держитесь той же линии: не уходить с площадки, не платить вперёд и не называть коды."
	tipDefaultDomain   = "Проверяйте домен: настоящий адрес заканчивается на avito.ru прямо перед первым слэшем."
)

func FallbackTips(mistakes []Mistake) []string {
	tips := make([]string, 0, len(mistakes)+1)
	seen := map[string]bool{}
	for _, m := range mistakes {
		t, ok := fallbackTips[m]
		if !ok || seen[t] {
			continue
		}
		seen[t] = true
		tips = append(tips, t)
	}
	if len(tips) == 0 {
		tips = append(tips, tipDefaultKeepLine, tipDefaultDomain)
	}
	if len(tips) > 5 {
		tips = tips[:5]
	}
	return tips
}
