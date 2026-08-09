import { Link } from 'react-router';

import { buttonVariants } from '@/components/ui/button';
import { ROUTES } from '@/configs/routes';
import { cn } from '@/lib/utils';

const NotFoundPage = () => {
  return (
    <div className="flex flex-col items-center gap-6 py-20 text-center">
      <p className="text-sm font-medium text-muted-foreground">404</p>
      <h1 className="text-3xl font-extrabold tracking-tight text-balance sm:text-4xl">
        Безопаша потерялся
      </h1>
      <p className="max-w-md text-muted-foreground text-balance">
        Такой страницы нет. Вернись на главную — там Безопаша тебя найдёт.
      </p>
      <Link
        to={ROUTES.root.create()}
        className={cn(buttonVariants({ size: 'lg' }))}
      >
        На главную
      </Link>
    </div>
  );
};

export default NotFoundPage;
