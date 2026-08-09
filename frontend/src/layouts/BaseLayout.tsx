import { Suspense } from 'react';
import { Outlet } from 'react-router';

import PageLoader from '@/shared/PageLoader';

import Header from './Header';

const BaseLayout = () => {
  return (
    <div className="h-svh overflow-y-auto">
      <div className="mx-auto flex h-full w-full max-w-3xl flex-col px-4 sm:px-6">
        <header className="shrink-0">
          <Header />
        </header>
        <main className="flex min-h-0 flex-1 flex-col">
          <Suspense fallback={<PageLoader />}>
            <Outlet />
          </Suspense>
        </main>
      </div>
    </div>
  );
};

export default BaseLayout;
