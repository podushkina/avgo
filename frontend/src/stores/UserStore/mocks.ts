import type { MeResponse } from './UserStore';

export const MOCK_TOTAL_TRAINING_STEPS = 2;

/** Mock GET /me — existing anonymous user. */
export const MOCK_ME_RESPONSE: MeResponse = {
  exists: true,
  user: {
    name: 'Алекс',
    age: '25',
    gender: 'male',
    buyer: {
      training: {
        currentStep: 0,
        totalSteps: MOCK_TOTAL_TRAINING_STEPS,
      },
      isExamPassed: false,
    },
    seller: {
      training: {
        currentStep: MOCK_TOTAL_TRAINING_STEPS,
        totalSteps: MOCK_TOTAL_TRAINING_STEPS,
      },
      isExamPassed: true,
    },
  },
};
