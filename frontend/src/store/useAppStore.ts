import { create } from 'zustand';
import type { AttemptResult, Difficulty, Role } from '../api/types';

export type View = 'home' | 'training' | 'results' | 'dialog' | 'progress';

interface AppState {
  userId: string | null;
  view: View;
  role: Role | null;
  difficulty: Difficulty;
  result: AttemptResult | null;
  theme: 'light' | 'dark';
  inviteSeen: boolean;

  setUserId: (id: string) => void;
  go: (view: View) => void;
  startTraining: (role: Role) => void;
  finishTraining: (result: AttemptResult) => void;
  startDialog: (role: Role, difficulty: Difficulty) => void;
  toggleTheme: () => void;
  dismissInvite: () => void;
}

const THEME_KEY = 'antifraud-theme';
const INVITE_KEY = 'antifraud-invite-seen';

export const storage = {
  get(key: string): string | null {
    try {
      return globalThis.localStorage?.getItem(key) ?? null;
    } catch {
      return null;
    }
  },
  set(key: string, value: string): void {
    try {
      globalThis.localStorage?.setItem(key, value);
    } catch {
      /* приватный режим браузера — работаем без сохранения */
    }
  },
};

function initialTheme(): 'light' | 'dark' {
  const stored = storage.get(THEME_KEY);
  if (stored === 'light' || stored === 'dark') return stored;
  return globalThis.matchMedia?.('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

export const useAppStore = create<AppState>((set) => ({
  userId: null,
  view: 'home',
  role: null,
  difficulty: 'medium',
  result: null,
  theme: initialTheme(),
  inviteSeen: storage.get(INVITE_KEY) === '1',

  setUserId: (id) => set({ userId: id }),
  go: (view) => set({ view }),

  startTraining: (role) => set({ role, view: 'training', result: null }),

  finishTraining: (result) =>
    set({ result, view: 'results', difficulty: result.suggested_difficulty }),

  startDialog: (role, difficulty) => set({ role, difficulty, view: 'dialog' }),

  toggleTheme: () =>
    set((state) => {
      const theme = state.theme === 'light' ? 'dark' : 'light';
      storage.set(THEME_KEY, theme);
      return { theme };
    }),

  dismissInvite: () => {
    storage.set(INVITE_KEY, '1');
    set({ inviteSeen: true });
  },
}));
