import { observer } from 'mobx-react-lite';
import { useEffect } from 'react';
import { Navigate, useNavigate, useParams } from 'react-router';

import PageLoader from '@/components/PageLoader';
import { Button } from '@/components/ui/button';
import { ROUTES } from '@/configs/routes';
import { HeaderRightSectionPortal } from '@/layouts';
import { isRole, rootStore } from '@/stores';

import ExamChat from './ExamChat';

const ExamPage = observer(() => {
  const { user: userStore, exam: examStore } = rootStore;
  const navigate = useNavigate();
  const { role: roleParam } = useParams();
  const role = isRole(roleParam) ? roleParam : null;

  useEffect(() => {
    if (!role) {
      return;
    }

    userStore.setRole(role);
    void examStore.start(role);
  }, [role, userStore, examStore]);

  if (!role) {
    return <Navigate to={ROUTES.root.create()} replace />;
  }

  const header = (
    <HeaderRightSectionPortal>
      <div className="flex flex-col items-end gap-0.5 text-right">
        <span className="text-sm font-semibold tracking-tight">Экзамен</span>
        {examStore.startStage.isSuccess ? (
          <span className="text-xs text-muted-foreground">
            Ход {examStore.cycle} из {examStore.maxCycles}
          </span>
        ) : null}
      </div>
    </HeaderRightSectionPortal>
  );

  if (examStore.startStage.isError) {
    return (
      <>
        {header}
        <div className="flex flex-1 flex-col items-center justify-center gap-4 py-8 text-center">
          <h1 className="text-xl font-semibold">Не удалось начать экзамен</h1>
          <p className="max-w-md text-sm text-muted-foreground">
            Собеседник сейчас недоступен. Проверь соединение и попробуй ещё раз.
          </p>
          <Button
            type="button"
            size="lg"
            onClick={() => void examStore.start(role)}
          >
            Попробовать снова
          </Button>
        </div>
      </>
    );
  }

  if (examStore.startStage.isLoading || !examStore.startStage.isSuccess) {
    return (
      <>
        {header}
        <PageLoader />
      </>
    );
  }

  const title =
    role === 'buyer'
      ? 'Безопаша хочет продать тебе товар — поговори с ним'
      : 'Безопаша хочет купить у тебя товар — поговори с ним';

  const handleSend = (text: string) => {
    void examStore.sendMessage(role, text);
  };

  const handleGoToResults = () => {
    if (examStore.isPassed) {
      userStore.markExamPassed(role);
    }

    void navigate(ROUTES.results.create(role));
  };

  const handleRestart = () => {
    void examStore.restart(role);
  };

  const handleFinish = () => {
    void examStore.finish(role);
  };

  return (
    <>
      {header}
      <div className="flex min-h-0 flex-1 flex-col gap-6 py-8">
        <h1 className="text-center text-2xl font-extrabold tracking-tight text-balance sm:text-3xl">
          {title}
        </h1>
        <ExamChat
          messages={examStore.messages}
          isWaitingReply={examStore.isWaitingReply}
          hasReplyError={examStore.replyStage.isError}
          isFinishing={examStore.isFinishing}
          verdict={examStore.verdict}
          explanation={examStore.explanation}
          cycle={examStore.cycle}
          maxCycles={examStore.maxCycles}
          progressPercent={examStore.progressPercent}
          onSend={handleSend}
          onGoToResults={handleGoToResults}
          onRestart={handleRestart}
          onFinish={handleFinish}
        />
      </div>
    </>
  );
});

export default ExamPage;
