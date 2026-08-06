import { useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { checkAnswer, fetchScenarios, submitAttempt } from '../api/client';
import type { CheckResult, Verdict } from '../api/types';
import { useAppStore } from '../store/useAppStore';

const verdictLabel: Record<Verdict, string> = {
  safe: 'Безопасное решение',
  risky: 'Рискованное решение',
  dangerous: 'Опасное решение',
};

export default function Training() {
  const { role, userId, finishTraining, go } = useAppStore();
  const [step, setStep] = useState(0);
  const [picked, setPicked] = useState<number | null>(null);
  const [feedback, setFeedback] = useState<CheckResult | null>(null);
  const [answers, setAnswers] = useState<{ scenario_id: number; option: number }[]>([]);

  const { data: scenarios, isPending } = useQuery({
    queryKey: ['scenarios', role],
    queryFn: () => fetchScenarios(role!),
    enabled: Boolean(role),
  });

  const check = useMutation({
    mutationFn: ({ id, option }: { id: number; option: number }) => checkAnswer(id, option),
    onSuccess: setFeedback,
  });

  const submit = useMutation({
    mutationFn: (all: { scenario_id: number; option: number }[]) =>
      submitAttempt(userId!, role!, all),
    onSuccess: finishTraining,
  });

  if (isPending || !scenarios) return <p className="muted">Загружаем сценарии…</p>;
  if (scenarios.length === 0) return <p className="muted">Сценарии не заведены.</p>;

  const scenario = scenarios[step];
  const isLast = step === scenarios.length - 1;
  const progress = ((step + (feedback ? 1 : 0)) / scenarios.length) * 100;

  function choose(index: number) {
    if (feedback) return;
    setPicked(index);
    setAnswers((prev) => [...prev, { scenario_id: scenario.id, option: index }]);
    check.mutate({ id: scenario.id, option: index });
  }

  function next() {
    if (isLast) {
      submit.mutate(answers);
      return;
    }
    setStep((s) => s + 1);
    setPicked(null);
    setFeedback(null);
  }

  function optionClass(index: number): string {
    if (!feedback) return 'option';
    if (index === picked) return `option option--picked-${feedback.your_verdict}`;
    if (index === feedback.correct_option) return 'option option--correct';
    return 'option';
  }

  return (
    <div className="card">
      <div className="row" style={{ justifyContent: 'space-between' }}>
        <span className="badge badge--info">
          {role === 'seller' ? 'Роль: продавец' : 'Роль: покупатель'}
        </span>
        <span className="muted">
          Ситуация {step + 1} из {scenarios.length}
        </span>
      </div>

      <div className="progressbar" style={{ margin: '14px 0 22px' }}>
        <div className="progressbar__fill" style={{ width: `${progress}%` }} />
      </div>

      <h2 className="h2">{scenario.title}</h2>
      <div className="situation">{scenario.situation}</div>

      <h3 className="h2" style={{ fontSize: 17, marginBottom: 14 }}>
        {scenario.question}
      </h3>

      {scenario.options.map((text, index) => (
        <button
          key={index}
          className={optionClass(index)}
          disabled={Boolean(feedback) || check.isPending}
          onClick={() => choose(index)}
        >
          {text}
        </button>
      ))}

      {feedback && (
        <>
          <div className={`outcome outcome--${feedback.your_verdict}`}>
            <span className="outcome__label">
              {verdictLabel[feedback.your_verdict]} · {feedback.points} из 10 очков
            </span>
            {feedback.your_outcome}
          </div>

          {!feedback.is_correct && (
            <div className="outcome outcome--safe">
              <span className="outcome__label">Как было бы безопасно</span>
              {feedback.correct_option_text}
              <p style={{ marginTop: 8 }}>{feedback.correct_outcome}</p>
            </div>
          )}

          <div className="card" style={{ background: 'var(--surface-2)', marginTop: 16 }}>
            <strong>Почему так</strong>
            <p className="muted" style={{ marginTop: 8 }}>
              {feedback.explanation}
            </p>
            <div className="flags">
              {feedback.red_flags.map((flag) => (
                <span key={flag} className="badge badge--dangerous">
                  ⚑ {flag}
                </span>
              ))}
            </div>
          </div>

          <div className="row" style={{ marginTop: 20 }}>
            <button className="btn btn--primary" onClick={next} disabled={submit.isPending}>
              {isLast ? 'Показать результат' : 'Следующая ситуация'}
            </button>
            <button className="btn btn--ghost" onClick={() => go('home')}>
              Прервать
            </button>
          </div>
        </>
      )}
    </div>
  );
}
