import { useEffect } from 'react';
import { useMutation } from '@tanstack/react-query';
import { ensureUser } from './api/client';
import { storage, useAppStore } from './store/useAppStore';
import InviteModal from './components/InviteModal';
import Home from './components/Home';
import Training from './components/Training';
import Results from './components/Results';
import Dialog from './components/Dialog';
import Progress from './components/Progress';

const EXTERNAL_ID_KEY = 'antifraud-external-id';

function externalId(): string {
  let id = storage.get(EXTERNAL_ID_KEY);
  if (!id) {
    id = crypto.randomUUID();
    storage.set(EXTERNAL_ID_KEY, id);
  }
  return id;
}

export default function App() {
  const { view, theme, userId, setUserId, go, toggleTheme } = useAppStore();

  const register = useMutation({
    mutationFn: () => ensureUser(externalId()),
    onSuccess: (user) => setUserId(user.id),
  });

  useEffect(() => {
    register.mutate();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
  }, [theme]);

  return (
    <div className="app">
      <header className="topbar">
        <div className="topbar__inner">
          <div className="logo">
            <span className="logo__mark">А</span>
            <span>Безопасная сделка</span>
          </div>
          <div className="topbar__spacer" />
          <button
            className={`navlink ${view === 'home' ? 'navlink--active' : ''}`}
            onClick={() => go('home')}
          >
            Тренажёр
          </button>
          <button
            className={`navlink ${view === 'progress' ? 'navlink--active' : ''}`}
            onClick={() => go('progress')}
          >
            Мой прогресс
          </button>
          <button className="navlink" onClick={toggleTheme} aria-label="Сменить тему">
            {theme === 'light' ? '🌙' : '☀️'}
          </button>
        </div>
      </header>

      <main className="page">
        {!userId && register.isPending && <p className="muted">Загружаем тренажёр…</p>}
        {register.isError && (
          <div className="card">
            <h2 className="h2">Сервис недоступен</h2>
            <p className="muted" style={{ marginTop: 8 }}>
              Не удалось связаться с бэкендом. Проверьте, что стек поднят: <code>make up</code>.
            </p>
          </div>
        )}

        {userId && (
          <>
            {view === 'home' && <Home />}
            {view === 'training' && <Training />}
            {view === 'results' && <Results />}
            {view === 'dialog' && <Dialog />}
            {view === 'progress' && <Progress />}
          </>
        )}
      </main>

      <InviteModal />
    </div>
  );
}
