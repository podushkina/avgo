import axios from 'axios';
import type {
  AttemptResult,
  CheckResult,
  DialogReport,
  DialogSession,
  Difficulty,
  ProgressEntry,
  Role,
  Scenario,
  User,
} from './types';

export const http = axios.create({ baseURL: '/api' });

export async function ensureUser(externalId: string): Promise<User> {
  const { data } = await http.post<User>('/users', { external_id: externalId });
  return data;
}

export async function fetchScenarios(role: Role): Promise<Scenario[]> {
  const { data } = await http.get<Scenario[]>('/scenarios', { params: { role } });
  return data;
}

export async function checkAnswer(scenarioId: number, option: number): Promise<CheckResult> {
  const { data } = await http.post<CheckResult>(`/scenarios/${scenarioId}/check`, { option });
  return data;
}

export async function submitAttempt(
  userId: string,
  role: Role,
  answers: { scenario_id: number; option: number }[],
): Promise<AttemptResult> {
  const { data } = await http.post<AttemptResult>('/attempts', {
    user_id: userId,
    role,
    answers,
  });
  return data;
}

export async function fetchProgress(userId: string): Promise<ProgressEntry[]> {
  const { data } = await http.get<ProgressEntry[]>('/progress', { params: { user_id: userId } });
  return data;
}

export async function createDialog(role: Role, difficulty: Difficulty): Promise<DialogSession> {
  const { data } = await http.post<DialogSession>('/dialog/sessions', { role, difficulty });
  return data;
}

export async function finishDialog(sessionId: string): Promise<{ report: DialogReport }> {
  const { data } = await http.post<{ report: DialogReport }>(
    `/dialog/sessions/${sessionId}/finish`,
    {},
  );
  return data;
}

export interface StreamHandlers {
  onToken: (text: string) => void;
  onDone: (full: string) => void;
  onError: (message: string) => void;
}

export async function streamReply(
  sessionId: string,
  text: string,
  handlers: StreamHandlers,
  signal?: AbortSignal,
): Promise<void> {
  const response = await fetch(`/api/dialog/sessions/${sessionId}/messages`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text }),
    signal,
  });

  if (!response.ok || !response.body) {
    const message = await response.text().catch(() => '');
    handlers.onError(message || `Сервис ответил ${response.status}`);
    return;
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const frames = buffer.split('\n\n');
    buffer = frames.pop() ?? '';

    for (const frame of frames) {
      const eventLine = frame.split('\n').find((l) => l.startsWith('event:'));
      const dataLine = frame.split('\n').find((l) => l.startsWith('data:'));
      if (!eventLine || !dataLine) continue;

      const event = eventLine.slice('event:'.length).trim();
      let payload: { text?: string; error?: string };
      try {
        payload = JSON.parse(dataLine.slice('data:'.length).trim());
      } catch {
        continue;
      }

      if (event === 'token' && payload.text) handlers.onToken(payload.text);
      else if (event === 'done') handlers.onDone(payload.text ?? '');
      else if (event === 'error') handlers.onError(payload.error ?? 'Неизвестная ошибка');
    }
  }
}
