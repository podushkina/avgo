import {
  CircleCheck,
  CircleX,
  GraduationCap,
  ShieldCheck,
  ThumbsUp,
  TriangleAlert,
} from 'lucide-react';
import { observer } from 'mobx-react-lite';
import { useEffect } from 'react';
import { Link, Navigate, useParams } from 'react-router';

import PageLoader from '@/components/PageLoader';
import { Button } from '@/components/ui/button';
import { ROUTES } from '@/configs/routes';
import { HeaderRightSectionPortal } from '@/layouts';
import { cn } from '@/lib/utils';
import { isRole, rootStore } from '@/stores';

const ResultsPage = observer(() => {
  const { user: userStore, results: resultsStore } = rootStore;
  const { role: roleParam } = useParams();
  const role = isRole(roleParam) ? roleParam : null;

  useEffect(() => {
    if (!role) {
      return;
    }

    userStore.setRole(role);
    void resultsStore.fetchResults(role);
  }, [role, userStore, resultsStore]);

  if (!role) {
    return <Navigate to={ROUTES.root.create()} replace />;
  }

  const header = (
    <HeaderRightSectionPortal>
      <div className="flex flex-col items-end gap-0.5 text-right">
        <span className="text-sm font-semibold tracking-tight">Результаты</span>
      </div>
    </HeaderRightSectionPortal>
  );

  const { results } = resultsStore;

  if (!results) {
    return (
      <>
        {header}
        {resultsStore.resultsStage.isError ? (
          <div className="flex flex-col items-center gap-4 py-16 text-center">
            <p className="text-muted-foreground">
              Не удалось загрузить результаты.
            </p>
            <Button
              type="button"
              size="lg"
              onClick={() => void resultsStore.fetchResults(role)}
            >
              Попробовать снова
            </Button>
          </div>
        ) : (
          <PageLoader />
        )}
      </>
    );
  }

  const isPassed = results.exam.verdict === 'passed';
  const roleLabel = role === 'buyer' ? 'покупателя' : 'продавца';

  return (
    <>
      {header}

      <div className="flex flex-col gap-6 py-8">
        <div className="flex flex-col items-center gap-2 text-center">
          <h1 className="text-2xl font-extrabold tracking-tight text-balance sm:text-3xl">
            Результаты обучения
          </h1>
          <p className="text-muted-foreground">Роль: {roleLabel}</p>
        </div>

        <div className="flex flex-col gap-4 rounded-2xl border bg-card p-4 sm:p-5">
          <div className="flex items-start gap-3">
            <GraduationCap className="mt-0.5 size-5 shrink-0 text-muted-foreground" />
            <div className="flex flex-col gap-1">
              <span className="font-semibold">Обучение</span>
              <span className="text-sm text-muted-foreground">
                Верных ответов: {results.training.correctSteps} из{' '}
                {results.training.totalSteps}
              </span>
            </div>
          </div>

          <div
            className={cn(
              'flex items-start gap-3 rounded-xl px-4 py-3',
              isPassed
                ? 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-400'
                : 'bg-destructive/10 text-destructive',
            )}
          >
            {isPassed ? (
              <CircleCheck className="mt-0.5 size-5 shrink-0" />
            ) : (
              <CircleX className="mt-0.5 size-5 shrink-0" />
            )}
            <div className="flex flex-col gap-1">
              <span className="font-semibold">
                {isPassed ? 'Экзамен сдан' : 'Экзамен не сдан'}
              </span>
              <span className="text-sm leading-relaxed text-foreground/80">
                {results.exam.explanation}
              </span>
            </div>
          </div>
        </div>

        {results.strengths?.length || results.weaknesses?.length ? (
          <div className="flex flex-col gap-4 rounded-2xl border bg-card p-4 sm:p-5">
            <span className="font-semibold">Разбор твоего разговора</span>

            {results.strengths?.length ? (
              <div className="flex flex-col gap-2">
                <span className="text-sm font-medium text-emerald-700 dark:text-emerald-400">
                  Сильные стороны
                </span>
                <ul className="flex flex-col gap-2">
                  {results.strengths.map((item) => (
                    <li
                      key={item}
                      className="flex gap-2 text-sm leading-relaxed text-muted-foreground"
                    >
                      <ThumbsUp className="mt-0.5 size-4 shrink-0 text-emerald-600 dark:text-emerald-400" />
                      {item}
                    </li>
                  ))}
                </ul>
              </div>
            ) : null}

            {results.weaknesses?.length ? (
              <div className="flex flex-col gap-2">
                <span className="text-sm font-medium text-amber-700 dark:text-amber-400">
                  Над чем поработать
                </span>
                <ul className="flex flex-col gap-2">
                  {results.weaknesses.map((item) => (
                    <li
                      key={item}
                      className="flex gap-2 text-sm leading-relaxed text-muted-foreground"
                    >
                      <TriangleAlert className="mt-0.5 size-4 shrink-0 text-amber-600 dark:text-amber-400" />
                      {item}
                    </li>
                  ))}
                </ul>
              </div>
            ) : null}
          </div>
        ) : null}

        <div className="flex flex-col gap-3 rounded-2xl border bg-card p-4 sm:p-5">
          <div className="flex items-center gap-3">
            <ShieldCheck className="size-5 shrink-0 text-muted-foreground" />
            <span className="font-semibold">Что запомнить</span>
          </div>
          <ul className="flex flex-col gap-2">
            {results.tips.map((tip) => (
              <li
                key={tip}
                className="flex gap-2 text-sm leading-relaxed text-muted-foreground"
              >
                <span aria-hidden className="text-foreground">
                  •
                </span>
                {tip}
              </li>
            ))}
          </ul>
        </div>

        <Button
          size="lg"
          className="w-full"
          render={<Link to={ROUTES.root.create()} />}
        >
          На главную
        </Button>
      </div>
    </>
  );
});

export default ResultsPage;
