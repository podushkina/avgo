import { makeAutoObservable } from 'mobx';

import { api } from '@/api';
import type { Role } from '../UserStore';
import { LoadingStageModel } from '../models';

export type ExamAuthor = 'user' | 'assistant';

export type ExamVerdict = 'passed' | 'failed';

export type ExamMessage = {
  id: string;
  author: ExamAuthor;
  text: string;
};

export type ExamReplyResponse = {
  message: string;

  /** Диалог закончен: модель больше не ждёт ответов. */
  isFinished: boolean;
  verdict: ExamVerdict | null;
  explanation: string | null;
  cycle?: number;
  maxCycles?: number;
};

type ExamStartResponse = {
  message: string;
  messages?: { id: number; author: 'scammer' | 'user'; text: string }[];
  isFinished?: boolean;
  verdict?: ExamVerdict | null;
  explanation?: string | null;
  cycle?: number;
  maxCycles?: number;
};

const DEFAULT_MAX_CYCLES = 6;

class ExamStore {
  messages: ExamMessage[] = [];
  verdict: ExamVerdict | null = null;
  explanation: string | null = null;
  cycle = 0;
  maxCycles = DEFAULT_MAX_CYCLES;

  startStage = new LoadingStageModel();
  replyStage = new LoadingStageModel();

  private _finished = false;
  isFinishing = false;

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
    return this._finished || this.verdict !== null;
  }

  get isPassed(): boolean {
    return this.verdict === 'passed';
  }

  get progressPercent(): number {
    if (this.maxCycles <= 0) {
      return 0;
    }

    return Math.min(100, Math.round((this.cycle / this.maxCycles) * 100));
  }

  async start(role: Role): Promise<void> {
    this.startStage.loading();
    this.messages = [];
    this.verdict = null;
    this.explanation = null;
    this._finished = false;
    this.replyStage.reset();

    try {
      const response = await api.get<ExamStartResponse>(
        `/exam/start?role=${role}`,
      );

      if (response.messages && response.messages.length > 0) {
        this.messages = response.messages.map((message) => ({
          id: String(message.id),
          author: message.author === 'user' ? 'user' : 'assistant',
          text: message.text,
        }));
      } else {
        this.messages = [
          { id: 'assistant-0', author: 'assistant', text: response.message },
        ];
      }

      this._finished = response.isFinished ?? false;
      this.verdict = response.verdict ?? null;
      this.explanation = response.explanation ?? null;
      this.cycle = response.cycle ?? 0;
      this.maxCycles = response.maxCycles ?? DEFAULT_MAX_CYCLES;
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
      const reply = await api.post<ExamReplyResponse>('/exam/message', {
        role,
        text: trimmed,
      });

      this.messages.push({
        id: `assistant-${this.messages.length}`,
        author: 'assistant',
        text: reply.message,
      });
      this._finished = reply.isFinished;
      this.verdict = reply.verdict;
      this.explanation = reply.explanation;
      this.cycle = reply.cycle ?? this.cycle + 1;
      this.maxCycles = reply.maxCycles ?? this.maxCycles;
      this.replyStage.success();
    } catch {
      this.replyStage.error();
    }
  }

  reset() {
    this.messages = [];
    this.verdict = null;
    this.explanation = null;
    this.cycle = 0;
    this.maxCycles = DEFAULT_MAX_CYCLES;
    this._finished = false;
    this.startStage.reset();
    this.replyStage.reset();
  }

  async finish(role: Role): Promise<void> {
    if (this.isFinished || this.replyStage.isLoading) {
      return;
    }

    this.isFinishing = true;
    this.replyStage.loading();

    try {
      const result = await api.post<{
        verdict: ExamVerdict;
        explanation: string;
      }>('/exam/finish', { role });

      this._finished = true;
      this.verdict = result.verdict;
      this.explanation = result.explanation;
      this.replyStage.success();
    } catch {
      this.replyStage.error();
    } finally {
      this.isFinishing = false;
    }
  }

  async restart(role: Role): Promise<void> {
    this.startStage.loading();
    this.messages = [];
    this.verdict = null;
    this.explanation = null;
    this.cycle = 0;
    this._finished = false;
    this.replyStage.reset();

    try {
      const response = await api.post<ExamStartResponse>('/exam/restart', {
        role,
      });

      this.messages = [
        { id: 'assistant-0', author: 'assistant', text: response.message },
      ];
      this.maxCycles = response.maxCycles ?? DEFAULT_MAX_CYCLES;
      this.startStage.success();
    } catch {
      this.startStage.error();
    }
  }
}

export default ExamStore;
