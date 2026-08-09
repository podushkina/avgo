import { makeAutoObservable } from 'mobx';

import type { ExamVerdict } from '../ExamStore';
import type { Role } from '../UserStore';
import { LoadingStageModel } from '../models';

import { MOCK_RESULTS } from './mocks';

export type ResultsResponse = {
  role: Role;
  training: {
    correctSteps: number;
    totalSteps: number;
  };
  exam: {
    verdict: ExamVerdict;
    explanation: string;
  };
  tips: string[];
};

const delay = (ms: number) =>
  new Promise<void>((resolve) => {
    setTimeout(resolve, ms);
  });

class ResultsStore {
  results: ResultsResponse | null = null;

  resultsStage = new LoadingStageModel();

  constructor() {
    makeAutoObservable(this, { resultsStage: false }, { autoBind: true });
  }

  get isPassed(): boolean {
    return this.results?.exam.verdict === 'passed';
  }

  async fetchResults(role: Role): Promise<void> {
    this.resultsStage.loading();
    this.results = null;

    try {
      await delay(700);

      // Mock POST /results { role } — anonymous user comes from the httpOnly cookie
      this.results = { ...MOCK_RESULTS[role] };
      this.resultsStage.success();
    } catch {
      this.resultsStage.error();
    }
  }

  reset() {
    this.results = null;
    this.resultsStage.reset();
  }
}

export default ResultsStore;
