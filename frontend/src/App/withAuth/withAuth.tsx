import { observer } from 'mobx-react-lite';
import type { ComponentType } from 'react';
import { Navigate } from 'react-router';

import { ROUTES } from '@/configs/routes';
import PageLoader from '@/shared/PageLoader';
import { rootStore } from '@/stores';

/** HOC для страниц, доступных только созданному пользователю. */
export const withAuth = <P extends Record<string, unknown>>(
  Component: ComponentType<P>,
) =>
  observer((props: P) => {
    const { exists, meStage } = rootStore.user;

    if (meStage.isLoading || meStage.isNotStarted) {
      return <PageLoader />;
    }

    if (meStage.isError || !exists) {
      return <Navigate to={ROUTES.root.create()} replace />;
    }

    return <Component {...props} />;
  });
