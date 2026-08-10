import { CircleCheck, CircleX, Send } from 'lucide-react';
import { useEffect, useRef, useState, type SubmitEvent } from 'react';

import LogoMark from '@/assets/logo-mark.svg?react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';
import type { ExamMessage, ExamVerdict } from '@/stores';

type ExamChatProps = {
  messages: ExamMessage[];
  isWaitingReply: boolean;
  hasReplyError: boolean;
  isFinishing: boolean;
  verdict: ExamVerdict | null;
  explanation: string | null;
  cycle: number;
  maxCycles: number;
  progressPercent: number;
  onSend: (text: string) => void;
  onGoToResults: () => void;
  onRestart: () => void;
  onFinish: () => void;
};

const ExamChat = ({
  messages,
  isWaitingReply,
  hasReplyError,
  isFinishing,
  verdict,
  explanation,
  cycle,
  maxCycles,
  progressPercent,
  onSend,
  onGoToResults,
  onRestart,
  onFinish,
}: ExamChatProps) => {
  const [text, setText] = useState('');
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ block: 'nearest' });
  }, [messages.length, isWaitingReply]);

  const isFinished = verdict !== null;
  const isPassed = verdict === 'passed';
  const canSend = text.trim() !== '' && !isWaitingReply && !isFinished;

  const handleSubmit = (event: SubmitEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (!canSend) {
      return;
    }

    onSend(text);
    setText('');
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-2xl border bg-card">
      <header className="flex items-center gap-3 border-b px-4 py-3 sm:px-5">
        <div
          className="flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-full bg-muted"
          aria-hidden
        >
          <LogoMark className="size-full scale-110" />
        </div>
        <div className="flex min-w-0 flex-1 flex-col">
          <span className="text-base font-semibold">Безопаша</span>
          <span className="text-xs text-muted-foreground">
            {isFinished ? 'Разговор завершён' : 'Правильного ответа нет'}
          </span>
        </div>
      </header>

      <div
        className="h-1 shrink-0 bg-muted"
        role="progressbar"
        aria-valuenow={cycle}
        aria-valuemin={0}
        aria-valuemax={maxCycles}
        aria-label="Прогресс экзамена"
      >
        <div
          className={cn(
            'h-full transition-all duration-500',
            isFinished && !isPassed ? 'bg-destructive' : 'bg-primary',
          )}
          style={{ width: `${isFinished ? 100 : progressPercent}%` }}
        />
      </div>

      <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto px-4 py-5 sm:px-5">
        {messages.map(({ id, author, text: messageText }) => (
          <div
            key={id}
            className={cn(
              'max-w-[85%] rounded-2xl px-4 py-3 text-sm leading-relaxed',
              author === 'assistant'
                ? 'rounded-tl-md bg-muted text-foreground'
                : 'self-end rounded-tr-md bg-primary text-primary-foreground',
            )}
          >
            {messageText}
          </div>
        ))}

        {isWaitingReply ? (
          <div
            role="status"
            aria-label={isFinishing ? 'Готовим разбор' : 'Безопаша печатает'}
            className="flex w-[70%] max-w-[85%] flex-col gap-2 rounded-2xl rounded-tl-md bg-muted px-4 py-3"
          >
            <span className="message-shimmer h-3 w-full rounded-full" />
            <span className="message-shimmer h-3 w-[80%] rounded-full" />
          </div>
        ) : null}

        <div ref={bottomRef} />
      </div>

      {isFinished ? (
        <div
          role="status"
          className={cn(
            'flex shrink-0 flex-col gap-4 border-t px-4 py-4 sm:px-5',
            isPassed
              ? 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-400'
              : 'bg-destructive/10 text-destructive',
          )}
        >
          <div className="flex items-start gap-3">
            {isPassed ? (
              <CircleCheck className="mt-0.5 size-5 shrink-0" />
            ) : (
              <CircleX className="mt-0.5 size-5 shrink-0" />
            )}
            <div className="flex flex-col gap-1">
              <span className="font-semibold">
                {isPassed ? 'Ты справился!' : 'Ты не справился'}
              </span>
              <span className="text-sm text-foreground/70">
                Завершено на ходе {cycle} из {maxCycles}
              </span>
              {explanation ? (
                <span className="text-sm leading-relaxed text-foreground/80">
                  {explanation}
                </span>
              ) : null}
            </div>
          </div>

          <div className="flex flex-col gap-2 sm:flex-row">
            <Button
              type="button"
              size="lg"
              variant="outline"
              className="w-full sm:flex-1"
              onClick={onRestart}
            >
              Пройти заново
            </Button>
            <Button
              type="button"
              size="lg"
              className="w-full sm:flex-1"
              onClick={onGoToResults}
            >
              К результатам
            </Button>
          </div>
        </div>
      ) : (
        <form
          onSubmit={handleSubmit}
          className="flex shrink-0 flex-col gap-2 border-t p-4 sm:p-5"
        >
          {hasReplyError ? (
            <p role="alert" className="text-xs text-destructive">
              Ответ не пришёл — собеседник недоступен. Попробуй отправить ещё
              раз.
            </p>
          ) : null}
          <div className="flex items-center gap-2">
            <Input
              value={text}
              onChange={(event) => setText(event.target.value)}
              placeholder={
                isFinishing
                  ? 'Готовим разбор разговора…'
                  : isWaitingReply
                    ? 'Безопаша печатает…'
                    : 'Напиши ответ Безопаше'
              }
              disabled={isWaitingReply}
              aria-label="Сообщение"
              className="h-10 flex-1"
            />
            <button
              type="submit"
              aria-label="Отправить"
              disabled={!canSend}
              className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground transition-colors outline-none hover:bg-primary/80 focus-visible:ring-3 focus-visible:ring-ring/50 disabled:pointer-events-none disabled:opacity-50"
            >
              <Send className="size-5" />
            </button>
          </div>
          <button
            type="button"
            onClick={onFinish}
            disabled={isWaitingReply}
            className="self-start text-xs text-muted-foreground underline-offset-4 transition-colors hover:text-foreground hover:underline disabled:pointer-events-none disabled:opacity-50"
          >
            Завершить разговор
          </button>
        </form>
      )}
    </div>
  );
};

export default ExamChat;
