import type { ExamReplyResponse, ExamVerdict } from '@/stores/ExamStore';
import type { Gender, MeUser, Role, RoleProgress } from '@/stores/UserStore';

import {
  EXAM_EXPLANATIONS,
  EXAM_FINAL_MESSAGES,
  EXAM_GREETINGS,
  EXAM_MESSAGES_LIMIT,
  EXAM_REPLIES,
  EXAM_RISKY_PATTERNS,
} from './data/exam';
import { RESULTS_BY_ROLE } from './data/results';
import { TRAINING_STEPS } from './data/training';
import {
  emptyRoleProgress,
  INITIAL_ME_RESPONSE,
  TOTAL_TRAINING_STEPS,
} from './data/user';

type ExamSession = {
  role: Role;
  replyIndex: number;
  isRiskyDialog: boolean;
  verdict: ExamVerdict | null;
  explanation: string | null;
};

type DbState = {
  exists: boolean;
  user: MeUser | null;
  exam: ExamSession | null;

  /** Completed training steps per role — internal mock pointer, not in API. */
  trainingStep: Record<Role, number>;

  /** Correct answers counted per role for results. */
  correctAnswers: Record<Role, number>;
};

const cloneProgress = (progress: RoleProgress): RoleProgress => ({
  status: progress.status,
});

const cloneUser = (user: MeUser): MeUser => ({
  name: user.name,
  age: user.age,
  gender: user.gender,
  buyer: cloneProgress(user.buyer),
  seller: cloneProgress(user.seller),
});

const isRiskyAnswer = (text: string) => {
  const normalized = text.toLowerCase();

  return EXAM_RISKY_PATTERNS.some((pattern) => normalized.includes(pattern));
};

const createInitialState = (): DbState => {
  const seed = INITIAL_ME_RESPONSE;

  return {
    exists: seed.exists,
    user: seed.user ? cloneUser(seed.user) : null,
    exam: null,
    trainingStep: {
      buyer: 0,
      seller:
        seed.user?.seller.status === 'exam_passed' ? TOTAL_TRAINING_STEPS : 0,
    },
    correctAnswers: { buyer: 0, seller: 0 },
  };
};

let state: DbState = createInitialState();

export const mockDb = {
  reset() {
    state = createInitialState();
  },

  getMe() {
    return {
      exists: state.exists,
      user: state.user ? cloneUser(state.user) : null,
    };
  },

  createUser(profile: { name: string; age: string; gender: Gender }) {
    const user: MeUser = {
      name: profile.name.trim(),
      age: profile.age,
      gender: profile.gender,
      buyer: emptyRoleProgress(),
      seller: emptyRoleProgress(),
    };

    state.exists = true;
    state.user = user;
    state.trainingStep = { buyer: 0, seller: 0 };
    state.correctAnswers = { buyer: 0, seller: 0 };
    state.exam = null;

    return { user: cloneUser(user) };
  },

  resetProgress(role: Role): RoleProgress {
    if (!state.user) {
      return emptyRoleProgress();
    }

    const progress = emptyRoleProgress();

    if (role === 'buyer') {
      state.user.buyer = progress;
    } else {
      state.user.seller = progress;
    }

    state.trainingStep[role] = 0;
    state.correctAnswers[role] = 0;

    if (state.exam?.role === role) {
      state.exam = null;
    }

    return cloneProgress(progress);
  },

  getProgress(role: Role): RoleProgress {
    if (!state.user) {
      return emptyRoleProgress();
    }

    return cloneProgress(
      role === 'buyer' ? state.user.buyer : state.user.seller,
    );
  },

  getTrainingStep(role: Role) {
    const steps = TRAINING_STEPS[role];
    const index = Math.min(state.trainingStep[role], steps.length - 1);
    const step = steps[index];

    return {
      currentStep: step.currentStep,
      totalSteps: step.totalSteps,
      productName: step.productName,
      message: step.message,
      variants: step.variants.map((variant) => ({ ...variant })),
    };
  },

  submitTrainingAnswer(role: Role, answerId: number) {
    const steps = TRAINING_STEPS[role];
    const index = Math.min(state.trainingStep[role], steps.length - 1);
    const step = steps[index];
    const isCorrect = answerId === step.correctAnswerId;

    if (isCorrect) {
      state.correctAnswers[role] += 1;
    }

    const nextStep =
      step.currentStep < step.totalSteps ? step.currentStep : step.totalSteps;

    state.trainingStep[role] = nextStep;

    if (state.user) {
      const roleProgress =
        role === 'buyer' ? state.user.buyer : state.user.seller;

      roleProgress.status =
        nextStep >= step.totalSteps
          ? 'training_passed'
          : 'training_in_progress';
    }

    return {
      isCorrect,
      explanation: step.explanation,
      currentStep: nextStep,
      totalSteps: step.totalSteps,
      isTrainingFinished: nextStep >= step.totalSteps,
    };
  },

  startExam(role: Role) {
    state.exam = {
      role,
      replyIndex: 0,
      isRiskyDialog: false,
      verdict: null,
      explanation: null,
    };

    if (state.user) {
      const progress = role === 'buyer' ? state.user.buyer : state.user.seller;

      progress.status = 'exam_in_progress';
    }

    return { message: EXAM_GREETINGS[role] };
  },

  examMessage(role: Role, text: string): ExamReplyResponse {
    if (state.exam?.role !== role) {
      state.exam = {
        role,
        replyIndex: 0,
        isRiskyDialog: false,
        verdict: null,
        explanation: null,
      };
    }

    const session = state.exam;

    if (isRiskyAnswer(text)) {
      session.isRiskyDialog = true;
    }

    session.replyIndex += 1;

    const isFinished =
      session.isRiskyDialog || session.replyIndex >= EXAM_MESSAGES_LIMIT;

    if (!isFinished) {
      const replies = EXAM_REPLIES[role];

      return {
        message: replies[(session.replyIndex - 1) % replies.length],
        isFinished: false,
        verdict: null,
        explanation: null,
      };
    }

    const verdict: ExamVerdict = session.isRiskyDialog ? 'failed' : 'passed';

    session.verdict = verdict;
    session.explanation = EXAM_EXPLANATIONS[verdict];

    if (state.user) {
      const progress = role === 'buyer' ? state.user.buyer : state.user.seller;

      progress.status = verdict === 'passed' ? 'exam_passed' : 'exam_failed';
    }

    return {
      message: EXAM_FINAL_MESSAGES[verdict],
      isFinished: true,
      verdict,
      explanation: EXAM_EXPLANATIONS[verdict],
    };
  },

  getResults(role: Role) {
    const base = RESULTS_BY_ROLE[role];
    const session = state.exam?.role === role ? state.exam : null;

    return {
      role,
      training: {
        correctSteps: state.correctAnswers[role],
        totalSteps: TOTAL_TRAINING_STEPS,
      },
      exam: {
        verdict: session?.verdict ?? base.exam.verdict,
        explanation: session?.explanation ?? base.exam.explanation,
      },
      tips: [...base.tips],
    };
  },
};
