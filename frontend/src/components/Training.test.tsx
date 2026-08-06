import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import Training from './Training';
import { useAppStore } from '../store/useAppStore';
import * as api from '../api/client';
import type { CheckResult, Scenario } from '../api/types';

const scenario: Scenario = {
  id: 1,
  role: 'seller',
  order_index: 1,
  title: 'Ссылка на получение оплаты',
  situation: 'Покупатель прислал ссылку.',
  question: 'Что вы сделаете?',
  options: ['Перейти по ссылке', 'Проверить заказ в приложении'],
};

const dangerous: CheckResult = {
  is_correct: false,
  your_verdict: 'dangerous',
  your_outcome: 'Данные карты уходят мошеннику.',
  points: 0,
  correct_option: 1,
  correct_option_text: 'Проверить заказ в приложении',
  correct_outcome: 'Заказа нет — вы ничего не потеряли.',
  explanation: 'Авито не присылает ссылки для получения денег.',
  red_flags: ['Ссылка на посторонний домен'],
};

function renderTraining() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <Training />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  vi.restoreAllMocks();
  useAppStore.setState({ role: 'seller', userId: 'u1', view: 'training', result: null });
  vi.spyOn(api, 'fetchScenarios').mockResolvedValue([scenario]);
});

describe('Training', () => {
  it('показывает ситуацию и варианты', async () => {
    renderTraining();

    expect(await screen.findByText('Ссылка на получение оплаты')).toBeInTheDocument();
    expect(screen.getByText('Покупатель прислал ссылку.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Перейти по ссылке' })).toBeInTheDocument();
  });

  it('после выбора показывает последствие, верный вариант и признаки риска', async () => {
    vi.spyOn(api, 'checkAnswer').mockResolvedValue(dangerous);
    const user = userEvent.setup();
    renderTraining();

    await user.click(await screen.findByRole('button', { name: 'Перейти по ссылке' }));

    expect(await screen.findByText('Данные карты уходят мошеннику.')).toBeInTheDocument();
    expect(screen.getByText('Заказа нет — вы ничего не потеряли.')).toBeInTheDocument();
    expect(screen.getByText(/Авито не присылает ссылки/)).toBeInTheDocument();
    expect(screen.getByText(/Ссылка на посторонний домен/)).toBeInTheDocument();
  });

  it('блокирует повторный ответ на тот же вопрос', async () => {
    const check = vi.spyOn(api, 'checkAnswer').mockResolvedValue(dangerous);
    const user = userEvent.setup();
    renderTraining();

    const option = await screen.findByRole('button', { name: 'Перейти по ссылке' });
    await user.click(option);
    await screen.findByText('Данные карты уходят мошеннику.');
    await user.click(screen.getByRole('button', { name: 'Проверить заказ в приложении' }));

    expect(check).toHaveBeenCalledTimes(1);
  });

  it('на последней ситуации отправляет попытку и уходит на результаты', async () => {
    vi.spyOn(api, 'checkAnswer').mockResolvedValue(dangerous);
    const submit = vi.spyOn(api, 'submitAttempt').mockResolvedValue({
      attempt_id: 'a1',
      role: 'seller',
      correct: 0,
      total: 1,
      percent: 0,
      score: 0,
      max_score: 10,
      level: 'Высокий риск',
      perfect: false,
      reviews: [],
      mistakes: [],
      missed_red_flags: [],
      suggested_difficulty: 'easy',
      completed_at: new Date().toISOString(),
    });
    const user = userEvent.setup();
    renderTraining();

    await user.click(await screen.findByRole('button', { name: 'Перейти по ссылке' }));
    await user.click(await screen.findByRole('button', { name: 'Показать результат' }));

    await waitFor(() =>
      expect(submit).toHaveBeenCalledWith('u1', 'seller', [{ scenario_id: 1, option: 0 }]),
    );
    await waitFor(() => expect(useAppStore.getState().view).toBe('results'));
  });
});
