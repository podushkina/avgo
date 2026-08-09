import { createBrowserRouter, Navigate } from 'react-router';

import { ROUTES } from '@/configs/routes';

import BaseLayout from '../layouts';

import {
  ExamPage,
  MainPage,
  NotFoundPage,
  ResultsPage,
  TrainingPage,
} from './lazyPages';

const router = createBrowserRouter([
  {
    path: ROUTES.root.path,
    element: <BaseLayout />,
    children: [
      {
        index: true,
        element: <MainPage />,
      },
      {
        path: ROUTES.training.path,
        element: <TrainingPage />,
      },
      {
        path: ROUTES.exam.path,
        element: <ExamPage />,
      },
      {
        path: ROUTES.results.path,
        element: <ResultsPage />,
      },
      {
        path: 'training',
        element: <Navigate to={ROUTES.root.create()} replace />,
      },
      {
        path: 'exam',
        element: <Navigate to={ROUTES.root.create()} replace />,
      },
      {
        path: 'results',
        element: <Navigate to={ROUTES.root.create()} replace />,
      },
      {
        path: '*',
        element: <NotFoundPage />,
      },
    ],
  },
]);

export default router;
