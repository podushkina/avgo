import { useQuery } from '@tanstack/react-query';
import { fetchProgress } from '../api/client';
import { useAppStore } from '../store/useAppStore';

const roleLabel = { seller: 'Продавец', buyer: 'Покупатель' } as const;

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString('ru-RU', {
    day: '2-digit',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export default function Progress() {
  const { userId, startTraining, go } = useAppStore();

  const { data, isPending } = useQuery({
    queryKey: ['progress', userId],
    queryFn: () => fetchProgress(userId!),
    enabled: Boolean(userId),
  });

  if (isPending) return <p className="muted">Загружаем историю…</p>;

  if (!data || data.length === 0) {
    return (
      <div className="card">
        <h2 className="h2">Прогресса пока нет</h2>
        <p className="muted" style={{ marginTop: 8, marginBottom: 18 }}>
          Пройдите первый тренинг — здесь появится история попыток и динамика результата.
        </p>
        <button className="btn btn--primary" onClick={() => go('home')}>
          Начать
        </button>
      </div>
    );
  }

  const chronological = [...data].reverse();
  const best = Math.max(...data.map((d) => d.percent));
  const latest = data[0];
  const first = chronological[0];
  const delta = Math.round((latest.percent - first.percent) * 10) / 10;

  return (
    <>
      <div className="card">
        <h2 className="h2">Мой прогресс</h2>
        <div className="row" style={{ marginTop: 16, gap: 28 }}>
          <div>
            <div className="score" style={{ fontSize: 32 }}>
              {best}%
            </div>
            <span className="muted">Лучший результат</span>
          </div>
          <div>
            <div className="score" style={{ fontSize: 32 }}>
              {data.length}
            </div>
            <span className="muted">Попыток</span>
          </div>
          <div>
            <div className="score" style={{ fontSize: 32 }}>
              {delta > 0 ? `+${delta}` : delta}
            </div>
            <span className="muted">Динамика, п.п.</span>
          </div>
        </div>

        <div className="spark" style={{ marginTop: 24 }}>
          {chronological.slice(-12).map((entry) => (
            <div
              key={entry.id}
              className="spark__bar"
              style={{ height: `${Math.max(entry.percent, 3)}%` }}
              title={`${entry.percent}% · ${formatDate(entry.completed_at)}`}
            />
          ))}
        </div>
        <p className="muted" style={{ marginTop: 8, fontSize: 13 }}>
          Последние попытки слева направо
        </p>
      </div>

      <div className="card">
        <h2 className="h2">История</h2>
        <div style={{ marginTop: 8 }}>
          {data.map((entry) => (
            <div key={entry.id} className="historyitem">
              <div>
                <strong>{roleLabel[entry.role]}</strong>
                <p className="muted" style={{ fontSize: 14 }}>
                  {formatDate(entry.completed_at)} · {entry.correct_count} из {entry.total_count} ·{' '}
                  {entry.score} очков
                </p>
              </div>
              <div className="row">
                <span className="badge badge--info">{entry.level}</span>
                <strong style={{ fontSize: 18 }}>{entry.percent}%</strong>
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="card">
        <h2 className="h2">Куда двигаться дальше</h2>
        <p className="muted" style={{ marginTop: 8, marginBottom: 18 }}>
          {best === 100
            ? 'Обе роли отработаны на максимум. Возвращайтесь к диалогу с ИИ на высокой сложности — там сценарий каждый раз новый.'
            : 'Повторное прохождение той же роли обычно даёт заметный прирост: признаки риска запоминаются после разбора.'}
        </p>
        <div className="row">
          <button className="btn btn--primary" onClick={() => startTraining('seller')}>
            Тренировка за продавца
          </button>
          <button className="btn btn--ghost" onClick={() => startTraining('buyer')}>
            Тренировка за покупателя
          </button>
        </div>
      </div>
    </>
  );
}
