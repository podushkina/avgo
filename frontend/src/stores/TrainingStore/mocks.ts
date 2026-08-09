import type { Role } from '../UserStore';

import type { TrainingStepResponse } from './TrainingStore';

/** Full mock step: client-facing fields + server-only answer check. */
export type MockTrainingStep = TrainingStepResponse & {
  correctAnswerId: number;
  explanation: string;
};

export const MOCK_TRAINING_STEPS: Record<Role, MockTrainingStep[]> = {
  buyer: [
    {
      currentStep: 1,
      totalSteps: 2,
      productName: 'смартфон',
      message:
        'Товар ещё в наличии, но желающих много. Переведи предоплату 2000 ₽ на карту, и я сниму объявление специально для тебя.',
      variants: [
        { id: 1, text: 'Доверять' },
        { id: 2, text: 'Не доверять' },
      ],
      correctAnswerId: 2,
      explanation:
        'Просьба о предоплате на карту вне защищённой сделки — классический признак мошенника.',
    },
    {
      currentStep: 2,
      totalSteps: 2,
      productName: 'ноутбук',
      message:
        'Могу привезти сегодня. Оплата только наличными при встрече у метро, без предоплаты.',
      variants: [
        { id: 1, text: 'Доверять' },
        { id: 2, text: 'Не доверять' },
      ],
      correctAnswerId: 1,
      explanation:
        'Встреча и оплата при получении без переводов на карту — нормальный сценарий покупки.',
    },
  ],
  seller: [
    {
      currentStep: 1,
      totalSteps: 2,
      productName: 'велосипед',
      message:
        'Я уже оплатил доставку. Перейди по ссылке и введи данные карты с CVV, чтобы деньги за товар зачислились тебе.',
      variants: [
        { id: 1, text: 'Доверять' },
        { id: 2, text: 'Не доверять' },
      ],
      correctAnswerId: 2,
      explanation:
        'Настоящая выплата никогда не требует ввода CVV по ссылке из переписки.',
    },
    {
      currentStep: 2,
      totalSteps: 2,
      productName: 'гитара',
      message:
        'Готов купить сегодня. Давай встретимся у торгового центра, посмотрю инструмент и оплачу на месте.',
      variants: [
        { id: 1, text: 'Доверять' },
        { id: 2, text: 'Не доверять' },
      ],
      correctAnswerId: 1,
      explanation:
        'Личная встреча и оплата на месте без подозрительных ссылок — безопасный сценарий продажи.',
    },
  ],
};
