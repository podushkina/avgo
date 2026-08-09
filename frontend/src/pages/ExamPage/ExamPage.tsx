import { observer } from 'mobx-react-lite';
import { useEffect } from 'react';
import { Navigate, useNavigate, useParams } from 'react-router';

import { ROUTES } from '@/configs/routes';
import { HeaderRightSectionPortal } from '@/layouts/Header';
import PageLoader from '@/shared/PageLoader';
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
      </div>
    </HeaderRightSectionPortal>
  );

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
    userStore.markExamPassed(role);
    void navigate(ROUTES.results.create(role));
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
          verdict={examStore.verdict}
          explanation={examStore.explanation}
          onSend={handleSend}
          onGoToResults={handleGoToResults}
        />
      </div>
    </>
  );
});

export default ExamPage;
