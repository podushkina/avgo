import type { MeResponse, RoleProgress } from '@/stores/UserStore';

export const TOTAL_TRAINING_STEPS = 2;

export const emptyRoleProgress = (): RoleProgress => ({
  status: 'not_started',
});

/** Seed for GET /api/me — existing anonymous user. */
export const INITIAL_ME_RESPONSE: MeResponse = {
  exists: true,
  user: {
    name: 'Алекс',
    age: '25',
    gender: 'male',
    buyer: {
      status: 'not_started',
    },
    seller: {
      status: 'exam_passed',
    },
  },
};
