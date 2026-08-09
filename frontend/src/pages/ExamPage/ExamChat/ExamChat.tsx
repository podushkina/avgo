import { CircleCheck, CircleX, Send } from 'lucide-react';
import { useEffect, useRef, useState, type FormEvent } from 'react';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';
import type { ExamMessage, ExamVerdict } from '@/stores';

type ExamChatProps = {
  messages: ExamMessage[];
  isWaitingReply: boolean;
  verdict: ExamVerdict | null;
  explanation: string | null;
  onSend: (text: string) => void;
  onGoToResults: () => void;
};

const ExamChat = ({
  messages,
  isWaitingReply,
  verdict,
  explanation,
  onSend,
  onGoToResults,
}: ExamChatProps) => {
  const [text, setText] = useState('');
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ block: 'nearest' });
  }, [messages.length, isWaitingReply]);

  const isFinished = verdict !== null;
  const isPassed = verdict === 'passed';
  const canSend = text.trim() !== '' && !isWaitingReply && !isFinished;

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
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
          className="flex size-10 shrink-0 items-center justify-center rounded-full bg-primary text-sm font-bold text-primary-foreground"
          aria-hidden
        >
          Б
        </div>
        <span className="text-base font-semibold">Безопаша</span>
      </header>

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
            aria-label="Безопаша печатает"
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
            onClick={onGoToResults}
          >
            Перейти к результатам обучения
          </Button>
        </div>
      ) : (
        <form
          onSubmit={handleSubmit}
          className="flex shrink-0 items-center gap-2 border-t p-4 sm:p-5"
        >
          <Input
            value={text}
            onChange={(event) => setText(event.target.value)}
            placeholder={
              isWaitingReply ? 'Безопаша печатает…' : 'Напиши ответ Безопаше'
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
        </form>
      )}
    </div>
  );
};

export default ExamChat;
