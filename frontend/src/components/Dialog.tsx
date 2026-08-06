import { useEffect, useRef, useState } from 'react';
import { createDialog, finishDialog, streamReply } from '../api/client';
import type { DialogReport } from '../api/types';
import { useAppStore } from '../store/useAppStore';

interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
}

export default function Dialog() {
  const { role, difficulty, go } = useAppStore();
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [draft, setDraft] = useState('');
  const [streaming, setStreaming] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [report, setReport] = useState<DialogReport | null>(null);
  const [turns, setTurns] = useState(0);
  const [maxTurns, setMaxTurns] = useState(20);
  const bottom = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!role) return;
    let cancelled = false;

    createDialog(role, difficulty)
      .then((session) => {
        if (cancelled) return;
        setSessionId(session.session_id);
        setMaxTurns(session.max_turns);
        setMessages([{ role: 'assistant', content: session.opening_message }]);
      })
      .catch(() => !cancelled && setError('Не удалось начать диалог'));

    return () => {
      cancelled = true;
    };
  }, [role, difficulty]);

  useEffect(() => {
    bottom.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  async function send() {
    const text = draft.trim();
    if (!text || !sessionId || streaming) return;

    setDraft('');
    setError(null);
    setStreaming(true);
    setTurns((t) => t + 1);
    setMessages((prev) => [
      ...prev,
      { role: 'user', content: text },
      { role: 'assistant', content: '' },
    ]);

    await streamReply(sessionId, text, {
      onToken: (chunk) =>
        setMessages((prev) => {
          const next = [...prev];
          next[next.length - 1] = {
            role: 'assistant',
            content: next[next.length - 1].content + chunk,
          };
          return next;
        }),
      onDone: (final) => {
        if (final) {
          setMessages((prev) => {
            const next = [...prev];
            next[next.length - 1] = { role: 'assistant', content: final };
            return next;
          });
        }
        setStreaming(false);
      },
      onError: (message) => {
        setError(message);
        setStreaming(false);
        setMessages((prev) => prev.slice(0, -1));
      },
    });
    setStreaming(false);
  }

  async function finish() {
    if (!sessionId) return;
    const { report: result } = await finishDialog(sessionId);
    setReport(result);
  }

  if (report) {
    return (
      <>
        <div className="card">
          <h2 className="h2">Разбор диалога</h2>
          <p className="muted" style={{ marginTop: 10 }}>
            {report.verdict}
          </p>
          <div className="row" style={{ marginTop: 14 }}>
            <span className={`badge ${report.survived ? 'badge--safe' : 'badge--dangerous'}`}>
              {report.survived ? '🛡 Вы ничего не отдали' : '⚠ Были уступки'}
            </span>
            <span className="badge badge--info">Реплик: {report.turns}</span>
          </div>
        </div>

        {report.tactics.length > 0 && (
          <div className="card">
            <h2 className="h2">Приёмы, которые применял собеседник</h2>
            {report.tactics.map((t) => (
              <div key={t.code} style={{ marginTop: 16 }}>
                <strong>{t.title}</strong>
                <p className="muted" style={{ marginTop: 4 }}>
                  {t.detail}
                </p>
                <div className="situation" style={{ margin: '8px 0 0' }}>
                  «{t.quote}»
                </div>
              </div>
            ))}
          </div>
        )}

        {report.mistakes.length > 0 && (
          <div className="card">
            <h2 className="h2">Что стоило сделать иначе</h2>
            {report.mistakes.map((m) => (
              <div key={m.code} className="outcome outcome--dangerous" style={{ marginTop: 12 }}>
                <span className="outcome__label">{m.title}</span>
                {m.detail}
              </div>
            ))}
          </div>
        )}

        <div className="card">
          <h2 className="h2">Запомните перед реальной сделкой</h2>
          <ul className="muted" style={{ marginTop: 10, paddingLeft: 20, lineHeight: 1.7 }}>
            {report.advice.map((a) => (
              <li key={a}>{a}</li>
            ))}
          </ul>
          <div className="row" style={{ marginTop: 20 }}>
            <button className="btn btn--primary" onClick={() => go('home')}>
              К тренажёру
            </button>
            <button className="btn btn--ghost" onClick={() => go('progress')}>
              Мой прогресс
            </button>
          </div>
        </div>
      </>
    );
  }

  return (
    <div className="card">
      <div className="banner">
        Это тренажёр. Ваш собеседник — языковая модель, играющая мошенника. Все ссылки и данные
        вымышленные, никому ничего не отправляйте по-настоящему.
      </div>

      <div className="row" style={{ justifyContent: 'space-between', marginBottom: 12 }}>
        <span className="badge badge--info">
          {role === 'seller' ? 'Вы продавец' : 'Вы покупатель'} · сложность: {difficulty}
        </span>
        <span className="muted">
          Реплик: {turns} из {maxTurns}
        </span>
      </div>

      <div className="chat">
        {messages.map((m, i) => (
          <div key={i} className={`bubble ${m.role === 'user' ? 'bubble--me' : 'bubble--them'}`}>
            {m.content || '…'}
          </div>
        ))}
        <div ref={bottom} />
      </div>

      {error && (
        <div className="outcome outcome--dangerous" style={{ marginTop: 12 }}>
          {error}
        </div>
      )}

      <div className="composer">
        <input
          value={draft}
          placeholder="Напишите ответ…"
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && send()}
          disabled={streaming || turns >= maxTurns}
        />
        <button
          className="btn btn--primary"
          onClick={send}
          disabled={streaming || !draft.trim() || turns >= maxTurns}
        >
          {streaming ? '…' : 'Отправить'}
        </button>
      </div>

      <div className="row" style={{ marginTop: 16 }}>
        <button className="btn btn--ghost" onClick={finish} disabled={turns === 0}>
          Завершить и получить разбор
        </button>
        <button className="btn btn--ghost" onClick={() => go('home')}>
          Выйти
        </button>
      </div>
    </div>
  );
}
