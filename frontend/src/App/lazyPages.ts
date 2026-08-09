import { lazy } from 'react';

import { withAuth } from './withAuth';

export const MainPage = lazy(() => import('@/pages/MainPage'));

export const NotFoundPage = lazy(() => import('@/pages/NotFoundPage'));

export const TrainingPage = lazy(async () => {
  const { default: Component } = await import('@/pages/TrainingPage');

  return { default: withAuth(Component) };
});

export const ExamPage = lazy(async () => {
  const { default: Component } = await import('@/pages/ExamPage');

  return { default: withAuth(Component) };
});

export const ResultsPage = lazy(async () => {
  const { default: Component } = await import('@/pages/ResultsPage');

  return { default: withAuth(Component) };
});
