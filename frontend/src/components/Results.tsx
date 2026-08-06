import { useAppStore } from '../store/useAppStore';

export default function Results() {
  const { result, role, go, startTraining, startDialog } = useAppStore();

  if (!result) {
    return <p className="muted">Результатов пока нет.</p>;
  }

  return (
    <>
      <div className="card">
        <div className="scoreline">
          <span className="score">{result.percent}%</span>
          <span className="badge badge--info">{result.level}</span>
          {result.perfect && <span className="badge badge--safe">🛡 Значок безопасной сделки</span>}
        </div>
        <p className="muted" style={{ marginTop: 10 }}>
          Верных решений: {result.correct} из {result.total}. Очков за безопасность: {result.score}{' '}
          из {result.max_score}.
        </p>

        {result.missed_red_flags.length > 0 && (
          <>
            <p style={{ marginTop: 20, fontWeight: 600 }}>Признаки, которые вы пропустили</p>
            <div className="flags">
              {result.missed_red_flags.map((flag) => (
                <span key={flag} className="badge badge--dangerous">
                  ⚑ {flag}
                </span>
              ))}
            </div>
          </>
        )}
      </div>

      {result.mistakes.length > 0 && (
        <div className="card">
          <h2 className="h2">Разбор ошибок</h2>
          {result.mistakes.map((m) => (
            <div key={m.scenario_id} style={{ marginTop: 20 }}>
              <strong>{m.title}</strong>
              <p className="muted" style={{ marginTop: 6 }}>
                {m.question}
              </p>
              <div className={`outcome outcome--${m.answered ? m.your_verdict : 'risky'}`}>
                <span className="outcome__label">
                  {m.answered ? `Вы выбрали: ${m.your_option_text}` : 'Вы не ответили'}
                </span>
                {m.answered ? m.your_outcome : 'Пропущенный вопрос засчитан как ошибка.'}
              </div>
              <div className="outcome outcome--safe">
                <span className="outcome__label">Безопасно: {m.correct_option_text}</span>
                {m.correct_outcome}
              </div>
              <p className="muted" style={{ marginTop: 12 }}>
                {m.explanation}
              </p>
            </div>
          ))}
        </div>
      )}

      <div className="card">
        <h2 className="h2">Что дальше</h2>
        <p className="muted" style={{ marginTop: 8, marginBottom: 18 }}>
          {result.percent >= 85
            ? 'Тест вы прошли уверенно — самое время проверить себя в живом диалоге на высокой сложности, где мошенник не выдаёт себя сразу.'
            : 'Теория усвоена не полностью. Попробуйте живой диалог: там мошенник импровизирует, и распознавать приёмы приходится по ходу разговора.'}
        </p>
        <div className="row">
          <button
            className="btn btn--primary"
            onClick={() => startDialog(result.role, result.suggested_difficulty)}
          >
            В диалог с ИИ ({result.suggested_difficulty})
          </button>
          <button className="btn btn--ghost" onClick={() => startTraining(result.role)}>
            Пройти заново
          </button>
          <button
            className="btn btn--ghost"
            onClick={() => startTraining(role === 'seller' ? 'buyer' : 'seller')}
          >
            Сменить роль
          </button>
          <button className="btn btn--ghost" onClick={() => go('progress')}>
            Мой прогресс
          </button>
        </div>
      </div>
    </>
  );
}
