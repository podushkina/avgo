import { makeAutoObservable } from 'mobx';

import type { Role } from '../UserStore';
import { LoadingStageModel } from '../models';

import { MOCK_TRAINING_STEPS } from './mocks';

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
};

const delay = (ms: number) =>
  new Promise<void>((resolve) => {
    setTimeout(resolve, ms);
  });

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

  async fetchCurrentStep(
    role: Role,
    progressCurrentStep: number,
  ): Promise<void> {
    this.stepStage.loading();
    this.selectedAnswerId = null;
    this.lastSubmit = null;
    this.answerStage.reset();

    try {
      await delay(500);

      // Mock GET /training/current-step?role={role}
      // Anonymous user is identified by the httpOnly cookie.
      // progress.currentStep: 0 = first step, totalSteps = finished.
      const steps = MOCK_TRAINING_STEPS[role];
      const index = Math.min(progressCurrentStep, steps.length - 1);
      const mockStep = steps[index];

      this.step = {
        currentStep: mockStep.currentStep,
        totalSteps: mockStep.totalSteps,
        productName: mockStep.productName,
        message: mockStep.message,
        variants: mockStep.variants.map((variant) => ({ ...variant })),
      };
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
      await delay(500);

      // Mock POST /training/answer { role, answer_id }
      const steps = MOCK_TRAINING_STEPS[role];
      const mockStep = steps.find(
        (step) => step.currentStep === this.step?.currentStep,
      );

      if (!mockStep) {
        throw new Error('Training step not found');
      }

      this.lastSubmit = {
        isCorrect: answerId === mockStep.correctAnswerId,
        explanation: mockStep.explanation,
      };

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
