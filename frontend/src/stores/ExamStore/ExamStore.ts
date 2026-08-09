import { makeAutoObservable } from 'mobx';

import type { Role } from '../UserStore';
import { LoadingStageModel } from '../models';

import {
  MOCK_EXAM_EXPLANATIONS,
  MOCK_EXAM_FINAL_MESSAGES,
  MOCK_EXAM_GREETINGS,
  MOCK_EXAM_MESSAGES_LIMIT,
  MOCK_EXAM_REPLIES,
  MOCK_EXAM_RISKY_PATTERNS,
} from './mocks';

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
};

const delay = (ms: number) =>
  new Promise<void>((resolve) => {
    setTimeout(resolve, ms);
  });

const isRiskyAnswer = (text: string) => {
  const normalized = text.toLowerCase();

  return MOCK_EXAM_RISKY_PATTERNS.some((pattern) =>
    normalized.includes(pattern),
  );
};

class ExamStore {
  messages: ExamMessage[] = [];
  verdict: ExamVerdict | null = null;
  explanation: string | null = null;

  startStage = new LoadingStageModel();
  replyStage = new LoadingStageModel();

  private _replyIndex = 0;
  private _isRiskyDialog = false;

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
    this._replyIndex = 0;
    this._isRiskyDialog = false;
    this.replyStage.reset();

    try {
      await delay(600);

      // Mock GET /exam/start?role={role}
      this.messages = [
        {
          id: 'assistant-0',
          author: 'assistant',
          text: MOCK_EXAM_GREETINGS[role],
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
      await delay(800);

      // Mock POST /exam/message { role, text } — reply comes from the AI model
      const reply = this._buildReply(role, trimmed);

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
    this._replyIndex = 0;
    this._isRiskyDialog = false;
    this.startStage.reset();
    this.replyStage.reset();
  }

  private _buildReply(role: Role, text: string): ExamReplyResponse {
    if (isRiskyAnswer(text)) {
      this._isRiskyDialog = true;
    }

    this._replyIndex += 1;

    const isFinished =
      this._isRiskyDialog || this._replyIndex >= MOCK_EXAM_MESSAGES_LIMIT;

    if (!isFinished) {
      const replies = MOCK_EXAM_REPLIES[role];

      return {
        message: replies[(this._replyIndex - 1) % replies.length],
        isFinished: false,
        verdict: null,
        explanation: null,
      };
    }

    const verdict: ExamVerdict = this._isRiskyDialog ? 'failed' : 'passed';

    return {
      message: MOCK_EXAM_FINAL_MESSAGES[verdict],
      isFinished: true,
      verdict,
      explanation: MOCK_EXAM_EXPLANATIONS[verdict],
    };
  }
}

export default ExamStore;
