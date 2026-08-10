import { Home } from 'lucide-react';
import { Link, useMatch } from 'react-router';

import Logo from '@/assets/logo.svg?react';
import { Button } from '@/components/ui/button';
import { ROUTES } from '@/configs/routes';

import { HEADER_RIGHT_SECTION_ID } from './constants';

const Header = () => {
  const isMainPage = useMatch(ROUTES.root.path) !== null;

  return (
    <div className="flex items-center justify-between gap-4 py-4">
      <Link to={ROUTES.root.create()} aria-label="На главную">
        <Logo className="h-10 w-auto max-w-[180px]" />
      </Link>

      <div className="flex items-center justify-end gap-3">
        <div id={HEADER_RIGHT_SECTION_ID} className="flex items-center" />

        {isMainPage ? null : (
          <Button
            variant="outline"
            size="lg"
            render={<Link to={ROUTES.root.create()} />}
          >
            <Home />
            На главную
          </Button>
        )}
      </div>
    </div>
  );
};

export default Header;
