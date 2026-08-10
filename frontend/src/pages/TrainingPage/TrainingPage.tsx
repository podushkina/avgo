import { observer } from 'mobx-react-lite';
import { useEffect } from 'react';
import { Navigate, useNavigate, useParams } from 'react-router';

import PageLoader from '@/components/PageLoader';
import { ROUTES } from '@/configs/routes';
import { HeaderRightSectionPortal } from '@/layouts';
import { isRole, rootStore } from '@/stores';

import Chat from './Chat';

const TrainingPage = observer(() => {
  const { user: userStore, training: trainingStore } = rootStore;
  const navigate = useNavigate();
  const { role: roleParam } = useParams();
  const role = isRole(roleParam) ? roleParam : null;

  useEffect(() => {
    if (!role) {
      return;
    }

    userStore.setRole(role);
    void trainingStore.fetchCurrentStep(role);
  }, [role, userStore, trainingStore]);

  if (!role) {
    return <Navigate to={ROUTES.root.create()} replace />;
  }

  const header = (
    <HeaderRightSectionPortal>
      <div className="flex flex-col items-end gap-0.5 text-right">
        <span className="text-sm font-semibold tracking-tight">Обучение</span>
        {trainingStore.step ? (
          <span className="text-xs text-muted-foreground">
            Шаг {trainingStore.step.currentStep} из{' '}
            {trainingStore.step.totalSteps}
          </span>
        ) : null}
      </div>
    </HeaderRightSectionPortal>
  );

  const { step } = trainingStore;

  if (!step || trainingStore.stepStage.isLoading) {
    return (
      <>
        {header}
        <PageLoader />
      </>
    );
  }

  const title =
    role === 'buyer'
      ? `Безопаша хочет продать тебе ${step.productName}`
      : `Безопаша хочет купить у тебя ${step.productName}`;

  const handleAnswer = (answerId: number) => {
    void trainingStore.submitAnswer(role, answerId).then(() => {
      if (!trainingStore.answerStage.isSuccess || !trainingStore.lastSubmit) {
        return;
      }

      userStore.setStatus(
        role,
        trainingStore.lastSubmit.isTrainingFinished
          ? 'training_passed'
          : 'training_in_progress',
      );
    });
  };

  const handleNext = () => {
    if (trainingStore.hasNextStep) {
      void trainingStore.fetchCurrentStep(role);

      return;
    }

    void navigate(ROUTES.exam.create(role));
  };

  return (
    <>
      {header}
      <div className="flex flex-col gap-6 py-8">
        <h1 className="text-center text-2xl font-extrabold tracking-tight text-balance sm:text-3xl">
          {title}
        </h1>
        <Chat
          key={step.currentStep}
          message={step.message}
          variants={step.variants}
          explanation={trainingStore.explanation}
          selectedAnswerId={trainingStore.selectedAnswerId}
          isSubmitting={trainingStore.answerStage.isLoading}
          isSubmitted={trainingStore.answerStage.isSuccess}
          isCorrect={trainingStore.isAnswerCorrect}
          hasNextStep={trainingStore.hasNextStep}
          isLoadingNext={trainingStore.stepStage.isLoading}
          onAnswer={handleAnswer}
          onNext={handleNext}
        />
      </div>
    </>
  );
});

export default TrainingPage;
