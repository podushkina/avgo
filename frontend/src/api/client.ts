const BASE_URL = '/api';

export type ApiErrorCode =
  | 'USER_NOT_FOUND'
  | 'TRAINING_NOT_PASSED'
  | 'TRAINING_ALREADY_PASSED'
  | 'STEP_MISMATCH'
  | 'INVALID_OPTION'
  | 'SESSION_NOT_FOUND'
  | 'SESSION_ALREADY_FINISHED'
  | 'MESSAGE_TOO_LONG'
  | 'RATE_LIMITED'
  | 'RESULTS_NOT_READY'
  | 'LLM_UNAVAILABLE'
  | 'BAD_REQUEST'
  | 'INTERNAL';

export class ApiError extends Error {
  readonly code: ApiErrorCode | (string & {});
  readonly status: number;
  readonly details?: unknown;

  constructor(
    status: number,
    code: string,
    message: string,
    details?: unknown,
  ) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.details = details;
  }
}

type ErrorEnvelope = {
  error?: { code?: string; message?: string; details?: unknown };
};

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;

  try {
    response = await fetch(BASE_URL + path, {
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      ...init,
    });
  } catch (cause) {
    throw new ApiError(0, 'NETWORK', 'Сервис недоступен', cause);
  }

  const text = await response.text();
  const body: unknown = text ? JSON.parse(text) : null;

  if (!response.ok) {
    const envelope = body as ErrorEnvelope | null;

    throw new ApiError(
      response.status,
      envelope?.error?.code ?? 'INTERNAL',
      envelope?.error?.message ?? 'Не удалось выполнить запрос',
      envelope?.error?.details,
    );
  }

  return body as T;
}

export const api = {
  get: <T>(path: string) => request<T>(path, { method: 'GET' }),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, {
      method: 'POST',
      body: body === undefined ? undefined : JSON.stringify(body),
    }),
};
