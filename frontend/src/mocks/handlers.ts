import { delay, http, HttpResponse } from 'msw';

import { isRole, type Gender, type Role } from '@/stores/UserStore';

import { mockDb } from './db';

const isGender = (value: unknown): value is Gender =>
  value === 'male' || value === 'female';

export const handlers = [
  http.get('/api/me', async () => {
    await delay(500);

    return HttpResponse.json(mockDb.getMe());
  }),

  http.post('/api/users', async ({ request }) => {
    await delay(400);

    const body = (await request.json()) as {
      name?: unknown;
      age?: unknown;
      gender?: unknown;
    };

    if (
      typeof body.name !== 'string' ||
      typeof body.age !== 'string' ||
      !isGender(body.gender)
    ) {
      return HttpResponse.json({ error: 'Invalid body' }, { status: 400 });
    }

    return HttpResponse.json(
      mockDb.createUser({
        name: body.name,
        age: body.age,
        gender: body.gender,
      }),
    );
  }),

  http.post('/api/progress/reset', async ({ request }) => {
    await delay(400);

    const body = (await request.json()) as { role?: unknown };

    if (!isRole(body.role)) {
      return HttpResponse.json({ error: 'Invalid role' }, { status: 400 });
    }

    return HttpResponse.json(mockDb.resetProgress(body.role));
  }),

  http.get('/api/training/current-step', async ({ request }) => {
    await delay(500);

    const url = new URL(request.url);
    const role = url.searchParams.get('role');

    if (!isRole(role)) {
      return HttpResponse.json({ error: 'Invalid role' }, { status: 400 });
    }

    return HttpResponse.json(mockDb.getTrainingStep(role));
  }),

  http.post('/api/training/answer', async ({ request }) => {
    await delay(500);

    const body = (await request.json()) as {
      role?: unknown;
      answer_id?: unknown;
    };

    if (!isRole(body.role) || typeof body.answer_id !== 'number') {
      return HttpResponse.json({ error: 'Invalid body' }, { status: 400 });
    }

    return HttpResponse.json(
      mockDb.submitTrainingAnswer(body.role, body.answer_id),
    );
  }),

  http.get('/api/exam/start', async ({ request }) => {
    await delay(600);

    const url = new URL(request.url);
    const roleParam = url.searchParams.get('role');

    if (!isRole(roleParam)) {
      return HttpResponse.json({ error: 'Invalid role' }, { status: 400 });
    }

    const role: Role = roleParam;

    return HttpResponse.json(mockDb.startExam(role));
  }),

  http.post('/api/exam/message', async ({ request }) => {
    await delay(800);

    const body = (await request.json()) as {
      role?: unknown;
      text?: unknown;
    };

    if (!isRole(body.role) || typeof body.text !== 'string') {
      return HttpResponse.json({ error: 'Invalid body' }, { status: 400 });
    }

    return HttpResponse.json(mockDb.examMessage(body.role, body.text));
  }),

  http.post('/api/results', async ({ request }) => {
    await delay(700);

    const body = (await request.json()) as { role?: unknown };

    if (!isRole(body.role)) {
      return HttpResponse.json({ error: 'Invalid role' }, { status: 400 });
    }

    return HttpResponse.json(mockDb.getResults(body.role));
  }),
];
