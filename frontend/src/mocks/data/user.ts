import type { MeResponse, RoleProgress } from '@/stores/UserStore';

export const TOTAL_TRAINING_STEPS = 2;

export const emptyRoleProgress = (): RoleProgress => ({
  training: {
    currentStep: 0,
    totalSteps: TOTAL_TRAINING_STEPS,
  },
  isExamPassed: false,
});

/** Seed for GET /api/me — existing anonymous user. */
export const INITIAL_ME_RESPONSE: MeResponse = {
  exists: true,
  user: {
    name: 'Алекс',
    age: '25',
    gender: 'male',
    buyer: {
      training: {
        currentStep: 0,
        totalSteps: TOTAL_TRAINING_STEPS,
      },
      isExamPassed: false,
    },
    seller: {
      training: {
        currentStep: TOTAL_TRAINING_STEPS,
        totalSteps: TOTAL_TRAINING_STEPS,
      },
      isExamPassed: true,
    },
  },
};
