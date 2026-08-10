import { Spinner } from '@/components/ui/spinner';

const PageLoader = () => {
  return (
    <div className="flex min-h-[50vh] items-center justify-center py-16">
      <Spinner className="size-8" />
    </div>
  );
};

export default PageLoader;
