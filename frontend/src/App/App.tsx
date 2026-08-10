import React, { useEffect } from 'react';
import { RouterProvider } from 'react-router';

import { rootStore } from '@/stores';

import router from './router';

const App: React.FC = () => {
  useEffect(() => {
    void rootStore.user.init();
  }, []);

  return <RouterProvider router={router} />;
};

export default App;
