import { makeAutoObservable } from 'mobx';

import { api } from '@/api';
import type { Role } from '../UserStore';
import { LoadingStageModel } from '../models';

export type TrainingVariant = {
  id: number;
  text: string;
};

export type TrainingStepResponse = {
  currentStep: number;
  totalSteps: number;
  productName: string;
  message: string;
  variants: TrainingVariant[];
};

export type SubmitAnswerResponse = {
  isCorrect: boolean;
  explanation: string;
  currentStep?: number;
  totalSteps?: number;
  isTrainingFinished?: boolean;
};

class TrainingStore {
  step: TrainingStepResponse | null = null;
  selectedAnswerId: number | null = null;
  lastSubmit: SubmitAnswerResponse | null = null;

  stepStage = new LoadingStageModel();
  answerStage = new LoadingStageModel();

  constructor() {
    makeAutoObservable(
      this,
      {
        stepStage: false,
        answerStage: false,
      },
      { autoBind: true },
    );
  }

  get hasStep(): boolean {
    return this.step !== null;
  }

  get hasNextStep(): boolean {
    if (!this.step) {
      return false;
    }

    return this.step.currentStep < this.step.totalSteps;
  }

  get isAnswerCorrect(): boolean | null {
    if (!this.lastSubmit) {
      return null;
    }

    return this.lastSubmit.isCorrect;
  }

  get explanation(): string | null {
    return this.lastSubmit?.explanation ?? null;
  }

  async fetchCurrentStep(role: Role): Promise<void> {
    this.stepStage.loading();
    this.selectedAnswerId = null;
    this.lastSubmit = null;
    this.answerStage.reset();

    try {
      this.step = await api.get<TrainingStepResponse>(
        `/training/current-step?role=${role}`,
      );
      this.stepStage.success();
    } catch {
      this.stepStage.error();
    }
  }

  async submitAnswer(role: Role, answerId: number): Promise<void> {
    if (!this.step) {
      return;
    }

    this.answerStage.loading();
    this.selectedAnswerId = answerId;

    try {
      this.lastSubmit = await api.post<SubmitAnswerResponse>(
        '/training/answer',
        {
          role,
          answer_id: answerId,
          stepNumber: this.step.currentStep,
        },
      );

      this.answerStage.success();
    } catch {
      this.answerStage.error();
    }
  }

  reset() {
    this.step = null;
    this.selectedAnswerId = null;
    this.lastSubmit = null;
    this.stepStage.reset();
    this.answerStage.reset();
  }
}

export default TrainingStore;
