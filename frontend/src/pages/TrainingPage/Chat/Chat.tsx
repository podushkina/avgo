import { CircleCheck, CircleX } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Spinner } from '@/components/ui/spinner';
import { cn } from '@/lib/utils';
import type { TrainingVariant } from '@/stores';

type ChatProps = {
  message: string;
  variants: TrainingVariant[];
  explanation: string | null;
  selectedAnswerId: number | null;
  isSubmitting: boolean;
  isSubmitted: boolean;
  isCorrect: boolean | null;
  hasNextStep: boolean;
  isLoadingNext: boolean;
  onAnswer: (answerId: number) => void;
  onNext: () => void;
};

const Chat = ({
  message,
  variants,
  explanation,
  selectedAnswerId,
  isSubmitting,
  isSubmitted,
  isCorrect,
  hasNextStep,
  isLoadingNext,
  onAnswer,
  onNext,
}: ChatProps) => {
  const isBusy = isSubmitting || isLoadingNext;

  return (
    <div className="flex flex-col overflow-hidden rounded-2xl border bg-card">
      <header className="flex items-center gap-3 border-b px-4 py-3 sm:px-5">
        <div
          className="flex size-10 shrink-0 items-center justify-center rounded-full bg-primary text-sm font-bold text-primary-foreground"
          aria-hidden
        >
          Б
        </div>
        <span className="text-base font-semibold">Безопаша</span>
      </header>

      <div className="flex flex-1 flex-col gap-3 px-4 py-5 sm:px-5">
        <div className="max-w-[85%] rounded-2xl rounded-tl-md bg-muted px-4 py-3 text-sm leading-relaxed text-foreground">
          {message}
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3 border-t p-4 sm:p-5">
        {variants.map((variant) => (
          <Button
            key={variant.id}
            type="button"
            variant={selectedAnswerId === variant.id ? 'default' : 'outline'}
            size="lg"
            disabled={isBusy || isSubmitted}
            onClick={() => onAnswer(variant.id)}
          >
            {isSubmitting && selectedAnswerId === variant.id ? (
              <Spinner className="size-4" />
            ) : null}
            {variant.text}
          </Button>
        ))}
      </div>

      {isSubmitted && isCorrect !== null ? (
        <div
          role="status"
          className={cn(
            'flex flex-col gap-4 border-t px-4 py-4 sm:px-5',
            isCorrect
              ? 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-400'
              : 'bg-destructive/10 text-destructive',
          )}
        >
          <div className="flex items-start gap-3">
            {isCorrect ? (
              <CircleCheck className="mt-0.5 size-5 shrink-0" />
            ) : (
              <CircleX className="mt-0.5 size-5 shrink-0" />
            )}
            <div className="flex flex-col gap-1">
              <span className="font-semibold">
                {isCorrect ? 'Правильно!' : 'Неправильно'}
              </span>
              {explanation ? (
                <span className="text-sm leading-relaxed text-foreground/80">
                  {explanation}
                </span>
              ) : null}
            </div>
          </div>

          <Button
            type="button"
            size="lg"
            className="w-full"
            disabled={isBusy}
            onClick={onNext}
          >
            {isLoadingNext ? <Spinner className="size-4" /> : null}
            {hasNextStep ? 'К следующему шагу' : 'К экзамену'}
          </Button>
        </div>
      ) : null}
    </div>
  );
};

export default Chat;
