package analysis

import (
	"regexp"
	"strings"

	"github.com/avito-antifraud/api/internal/llm"
)

type Finding struct {
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Quote  string `json:"quote"`
}

type Report struct {
	Tactics  []Finding `json:"tactics"`
	Mistakes []Finding `json:"mistakes"`
	Turns    int       `json:"turns"`
	Survived bool      `json:"survived"`
	Verdict  string    `json:"verdict"`
	Advice   []string  `json:"advice"`
}

type rule struct {
	code   string
	title  string
	detail string
	re     *regexp.Regexp
}

var tacticRules = []rule{
	{"phishing_link", "Фишинговая ссылка",
		"Собеседник предлагал перейти на сторонний сайт для оплаты или подтверждения.",
		regexp.MustCompile(`(?i)\S+\.(example|ru|com|net|org)\b|ссылк|перейдит|перейди|нажмит`)},
	{"urgency", "Давление срочностью",
		"Вас торопили, чтобы вы не успели проверить детали.",
		regexp.MustCompile(`(?i)срочн|быстре|скоре|тороп|10 минут|сгор|успе[йе]|прямо сейчас|опазд`)},
	{"card_data", "Запрос данных карты",
		"У вас пытались получить реквизиты карты, включая код CVC.",
		regexp.MustCompile(`(?i)карт[ыуои]|cvc|cvv|трёхзначн|трехзначн|реквизит`)},
	{"sms_code", "Запрос кода из СМС",
		"У вас просили одноразовый код — это попытка захватить аккаунт или счёт.",
		regexp.MustCompile(`(?i)код из смс|код из сообщ|назовите код|продиктуйте код|код подтвержд|пришёл код|пришел код`)},
	{"off_platform", "Увод с площадки",
		"Вам предлагали продолжить общение вне платформы, где нет защиты сделки.",
		regexp.MustCompile(`(?i)whatsapp|вотсап|ватсап|telegram|телеграм|вайбер|viber|напишите мне на`)},
	{"prepay", "Требование предоплаты",
		"Деньги просили вперёд, вне защищённой сделки.",
		regexp.MustCompile(`(?i)предоплат|аванс|задаток|бронь|забронир|переведите|перевод на карт`)},
	{"authority", "Подмена лица площадки",
		"Собеседник выдавал себя за сервис, поддержку или службу безопасности.",
		regexp.MustCompile(`(?i)служба безопасн|поддержк|модерац|администрац|сервис требует|система требует`)},
	{"overpay", "Схема с переплатой",
		"Предлагали перевести больше и вернуть разницу — классическая возвратная схема.",
		regexp.MustCompile(`(?i)больше.{0,30}верн|верн.{0,30}разниц|переплат|лишн.{0,20}верн`)},
}

var mistakeRules = []rule{
	{"shared_card", "Вы назвали номер карты",
		"В переписке появился номер карты. Реквизиты нельзя отправлять собеседнику.",
		regexp.MustCompile(`\b\d{4}[ -]?\d{4}[ -]?\d{4}[ -]?\d{4}\b`)},
	{"shared_cvc", "Вы назвали код с обратной стороны карты",
		"CVC нужен только для списания денег. Назвав его, вы отдали доступ к счёту.",
		regexp.MustCompile(`(?i)cvc|cvv|(код|цифр\w*)\s+(с|на)\s+обратной`)},
	{"shared_code", "Вы передали одноразовый код",
		"Код из СМС подтверждает действия с вашим аккаунтом и не сообщается никому.",
		regexp.MustCompile(`(?i)(код|пароль)\D{0,20}\d{4,6}|\bмой код\b|вот код`)},
	{"followed_link", "Вы перешли по ссылке",
		"Переход на сторонний сайт — это то, ради чего затевалась вся переписка.",
		regexp.MustCompile(`(?i)переш[её]л|перешла|открыл ссылк|зашёл на сайт|зашел на сайт|ввёл данн|ввел данн|заполнил форм`)},
	{"agreed_prepay", "Вы согласились заплатить вперёд",
		"Предоплата незнакомому человеку не защищена и не возвращается.",
		regexp.MustCompile(`(?i)перевед[уё]|переводу|оплачу|оплатил|скину денег|отправил деньги|готов внести|внесу предоплат`)},
	{"shared_personal", "Вы раскрыли личные данные",
		"Телефон, паспорт и адрес используются для оформления займов и взлома аккаунтов.",
		regexp.MustCompile(`(?i)\+7\d{10}|\b8\d{10}\b|паспорт|снилс|адрес прожив`)},
	{"went_off_platform", "Вы согласились уйти с площадки",
		"Вне платформы не работают защита сделки, модерация и поддержка.",
		regexp.MustCompile(`(?i)(да|хорошо|ок|давайте|можно)\W{0,15}(whatsapp|вотсап|ватсап|telegram|телеграм)|мой (ватсап|телеграм|номер)`)},
}

const quoteLimit = 160

func Analyze(msgs []llm.Message) Report {
	rep := Report{Tactics: []Finding{}, Mistakes: []Finding{}, Advice: []string{}}

	seenTactic := map[string]bool{}
	seenMistake := map[string]bool{}

	for _, m := range msgs {
		switch m.Role {
		case llm.RoleAssistant:
			collect(tacticRules, m.Content, seenTactic, &rep.Tactics)
		case llm.RoleUser:
			rep.Turns++
			collect(mistakeRules, m.Content, seenMistake, &rep.Mistakes)
		}
	}

	rep.Survived = len(rep.Mistakes) == 0
	rep.Verdict = verdictFor(rep)
	rep.Advice = adviceFor(rep)
	return rep
}

func collect(rules []rule, text string, seen map[string]bool, out *[]Finding) {
	for _, r := range rules {
		if seen[r.code] || !r.re.MatchString(text) {
			continue
		}
		seen[r.code] = true
		*out = append(*out, Finding{
			Code:   r.code,
			Title:  r.title,
			Detail: r.detail,
			Quote:  quote(text),
		})
	}
}

func quote(text string) string {
	t := strings.TrimSpace(text)
	if len(t) <= quoteLimit {
		return t
	}
	runes := []rune(t)
	if len(runes) <= quoteLimit {
		return t
	}
	return strings.TrimSpace(string(runes[:quoteLimit])) + "…"
}

func verdictFor(rep Report) string {
	switch {
	case rep.Turns == 0:
		return "Диалог не начался — попробуйте пообщаться с собеседником и довести ситуацию до развязки."
	case rep.Survived && rep.Turns >= 5:
		return "Вы выдержали давление и не отдали ничего ценного. Именно так выглядит безопасное поведение в реальной сделке."
	case rep.Survived:
		return "Пока всё чисто, но диалог был коротким. Попробуйте продолжить — давление обычно начинается позже."
	case len(rep.Mistakes) == 1:
		return "Одна уступка — и этого достаточно, чтобы потерять деньги или аккаунт. Разберите её ниже."
	default:
		return "Собеседник получил от вас сразу несколько вещей, которых не должен был получить. Разберите каждую ниже."
	}
}

func adviceFor(rep Report) []string {
	advice := []string{}
	has := func(code string) bool {
		for _, f := range rep.Mistakes {
			if f.Code == code {
				return true
			}
		}
		return false
	}

	if has("shared_code") || has("shared_cvc") {
		advice = append(advice, "Одноразовые коды и CVC не сообщаются никому и ни в какой части — ни продавцу, ни «поддержке».")
	}
	if has("shared_card") {
		advice = append(advice, "Для зачисления денег достаточно номера карты, но безопаснее вообще не отправлять реквизиты в чат.")
	}
	if has("followed_link") {
		advice = append(advice, "Статус заказа и оплату проверяйте только внутри приложения, а не по ссылке из переписки.")
	}
	if has("agreed_prepay") {
		advice = append(advice, "Платите при получении или через сделку на площадке — предоплата переводом не возвращается.")
	}
	if has("went_off_platform") {
		advice = append(advice, "Оставайтесь в чате площадки: вне его не работают ни защита сделки, ни поддержка.")
	}
	if has("shared_personal") {
		advice = append(advice, "Паспорт и телефон не нужны для покупки или продажи — по их фото оформляют займы.")
	}
	if len(advice) == 0 {
		advice = append(advice, "Держитесь той же линии: не уходить с площадки, не платить вперёд и не называть коды.")
	}
	return advice
}
