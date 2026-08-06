import { describe, expect, it, vi, afterEach } from 'vitest';
import { streamReply } from './client';

function sseResponse(frames: string[], ok = true, status = 200): Response {
  const body = new ReadableStream<Uint8Array>({
    start(controller) {
      const encoder = new TextEncoder();
      for (const frame of frames) controller.enqueue(encoder.encode(frame));
      controller.close();
    },
  });
  return { ok, status, body, text: async () => '' } as unknown as Response;
}

function token(text: string): string {
  return `event: token\ndata: ${JSON.stringify({ text })}\n\n`;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('streamReply', () => {
  it('собирает токены и отдаёт итоговый текст', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValue(
          sseResponse([
            token('Здрав'),
            token('ствуйте'),
            `event: done\ndata: ${JSON.stringify({ text: 'Здравствуйте' })}\n\n`,
          ]),
        ),
    );

    const chunks: string[] = [];
    const onDone = vi.fn();

    await streamReply('s1', 'привет', {
      onToken: (t) => chunks.push(t),
      onDone,
      onError: vi.fn(),
    });

    expect(chunks.join('')).toBe('Здравствуйте');
    expect(onDone).toHaveBeenCalledWith('Здравствуйте');
  });

  it('склеивает кадры, разорванные между чанками', async () => {
    const frame = token('Привет');
    const cut = Math.floor(frame.length / 2);

    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValue(
          sseResponse([frame.slice(0, cut), frame.slice(cut), 'event: done\ndata: {}\n\n']),
        ),
    );

    const chunks: string[] = [];
    await streamReply('s1', 'x', {
      onToken: (t) => chunks.push(t),
      onDone: vi.fn(),
      onError: vi.fn(),
    });

    expect(chunks.join('')).toBe('Привет');
  });

  it('сообщает об ошибке из потока', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValue(
          sseResponse([
            `event: error\ndata: ${JSON.stringify({ error: 'модель недоступна' })}\n\n`,
          ]),
        ),
    );

    const onError = vi.fn();
    await streamReply('s1', 'x', { onToken: vi.fn(), onDone: vi.fn(), onError });

    expect(onError).toHaveBeenCalledWith('модель недоступна');
  });

  it('сообщает об ошибке при неуспешном HTTP-статусе', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(sseResponse([], false, 503)));

    const onError = vi.fn();
    const onToken = vi.fn();
    await streamReply('s1', 'x', { onToken, onDone: vi.fn(), onError });

    expect(onError).toHaveBeenCalled();
    expect(onToken).not.toHaveBeenCalled();
  });

  it('пропускает кадры с некорректным JSON, не падая', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValue(
          sseResponse([
            'event: token\ndata: {сломано\n\n',
            token('ок'),
            'event: done\ndata: {}\n\n',
          ]),
        ),
    );

    const chunks: string[] = [];
    await streamReply('s1', 'x', {
      onToken: (t) => chunks.push(t),
      onDone: vi.fn(),
      onError: vi.fn(),
    });

    expect(chunks.join('')).toBe('ок');
  });
});
