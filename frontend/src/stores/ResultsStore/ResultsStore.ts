import { makeAutoObservable } from 'mobx';

import { api } from '@/api';
import type { ExamVerdict } from '../ExamStore';
import type { Role } from '../UserStore';
import { LoadingStageModel } from '../models';

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
  strengths?: string[];
  weaknesses?: string[];
  score?: number;
  grade?: string;
};

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
      this.results = await api.post<ResultsResponse>('/results', { role });
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
