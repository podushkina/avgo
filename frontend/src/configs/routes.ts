import type { Role } from '@/stores';

export const ROUTES = {
  root: {
    path: '/',
    create: () => '/',
  },
  training: {
    path: '/training/:role',
    create: (role: Role) => `/training/${role}`,
  },
  exam: {
    path: '/exam/:role',
    create: (role: Role) => `/exam/${role}`,
  },
  results: {
    path: '/results/:role',
    create: (role: Role) => `/results/${role}`,
  },
} as const;
