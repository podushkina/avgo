import { makeAutoObservable } from 'mobx';

import { api } from '@/shared/api';

import type { Role } from '../UserStore';
import { LoadingStageModel } from '../models';

export type ExamAuthor = 'user' | 'assistant';

export type ExamVerdict = 'passed' | 'failed';

export type ExamMessage = {
  id: string;
  author: ExamAuthor;
  text: string;
};

export type ExamStartResponse = {
  message: string;
};

export type ExamReplyResponse = {
  message: string;

  /** Диалог закончен: модель больше не ждёт ответов. */
  isFinished: boolean;
  verdict: ExamVerdict | null;
  explanation: string | null;
};

class ExamStore {
  messages: ExamMessage[] = [];
  verdict: ExamVerdict | null = null;
  explanation: string | null = null;

  startStage = new LoadingStageModel();
  replyStage = new LoadingStageModel();

  constructor() {
    makeAutoObservable(
      this,
      {
        startStage: false,
        replyStage: false,
      },
      { autoBind: true },
    );
  }

  get isWaitingReply(): boolean {
    return this.replyStage.isLoading;
  }

  get isFinished(): boolean {
    return this.verdict !== null;
  }

  get isPassed(): boolean {
    return this.verdict === 'passed';
  }

  async start(role: Role): Promise<void> {
    this.startStage.loading();
    this.messages = [];
    this.verdict = null;
    this.explanation = null;
    this.replyStage.reset();

    try {
      const response = await api.get<ExamStartResponse>(
        `/api/exam/start?role=${role}`,
      );

      this.messages = [
        {
          id: 'assistant-0',
          author: 'assistant',
          text: response.message,
        },
      ];
      this.startStage.success();
    } catch {
      this.startStage.error();
    }
  }

  async sendMessage(role: Role, text: string): Promise<void> {
    const trimmed = text.trim();

    if (trimmed === '' || this.replyStage.isLoading || this.isFinished) {
      return;
    }

    this.messages.push({
      id: `user-${this.messages.length}`,
      author: 'user',
      text: trimmed,
    });

    this.replyStage.loading();

    try {
      const reply = await api.post<ExamReplyResponse>('/api/exam/message', {
        role,
        text: trimmed,
      });

      this.messages.push({
        id: `assistant-${this.messages.length}`,
        author: 'assistant',
        text: reply.message,
      });
      this.verdict = reply.verdict;
      this.explanation = reply.explanation;
      this.replyStage.success();
    } catch {
      this.replyStage.error();
    }
  }

  reset() {
    this.messages = [];
    this.verdict = null;
    this.explanation = null;
    this.startStage.reset();
    this.replyStage.reset();
  }
}

export default ExamStore;
