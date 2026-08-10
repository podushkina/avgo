import { makeAutoObservable } from 'mobx';

import { api } from '@/api';

import { LoadingStageModel } from '../models';

export type Role = 'buyer' | 'seller';

export type Gender = 'male' | 'female';

export type ProgressStatus =
  | 'not_started'
  | 'training_in_progress'
  | 'training_passed'
  | 'exam_in_progress'
  | 'exam_passed'
  | 'exam_failed';

export type RoleProgress = {
  status: ProgressStatus;
};

export type MeUser = {
  name: string;
  age: string;
  gender: Gender;
  buyer: RoleProgress;
  seller: RoleProgress;
};

export type MeResponse = {
  exists: boolean;
  user: MeUser | null;
};

export type UserProfile = {
  name: string;
  age: string;
  gender: Gender;
};

export const isRole = (value: unknown): value is Role =>
  value === 'buyer' || value === 'seller';

export const isTrainingPassed = (status: ProgressStatus): boolean =>
  status === 'training_passed' ||
  status === 'exam_in_progress' ||
  status === 'exam_passed' ||
  status === 'exam_failed';

export const isExamFinished = (status: ProgressStatus): boolean =>
  status === 'exam_passed' || status === 'exam_failed';

export const hasRoleProgress = (progress: RoleProgress): boolean =>
  progress.status !== 'not_started';

const emptyProgress = (): RoleProgress => ({
  status: 'not_started',
});

const cloneProgress = (progress: RoleProgress): RoleProgress => ({
  status: progress.status,
});

class UserStore {
  exists = false;
  name = '';
  age = '';
  gender: Gender | null = null;
  role: Role = 'buyer';
  buyer: RoleProgress = emptyProgress();
  seller: RoleProgress = emptyProgress();

  meStage = new LoadingStageModel();
  submitStage = new LoadingStageModel();
  resetStage = new LoadingStageModel();

  constructor() {
    makeAutoObservable(
      this,
      {
        meStage: false,
        submitStage: false,
        resetStage: false,
      },
      { autoBind: true },
    );
  }

  get hasProfile(): boolean {
    return this.name.trim() !== '' && this.age !== '' && this.gender !== null;
  }

  getProgress(role: Role): RoleProgress {
    return role === 'buyer' ? this.buyer : this.seller;
  }

  setRole(role: Role) {
    this.role = role;
  }

  setStatus(role: Role, status: ProgressStatus) {
    this.getProgress(role).status = status;
  }

  applyMeResponse({ exists, user }: MeResponse) {
    this.exists = exists;

    if (!exists || !user) {
      this.name = '';
      this.age = '';
      this.gender = null;
      this.buyer = emptyProgress();
      this.seller = emptyProgress();

      return;
    }

    this.name = user.name;
    this.age = user.age;
    this.gender = user.gender;
    this.buyer = cloneProgress(user.buyer);
    this.seller = cloneProgress(user.seller);
  }

  setProfile({ name, age, gender }: UserProfile) {
    this.name = name.trim();
    this.age = age;
    this.gender = gender;
    this.submitStage.reset();
  }

  /** Загружает профиль один раз за жизнь приложения; повторные вызовы игнорируются. */
  async init(): Promise<void> {
    if (!this.meStage.isNotStarted) {
      return;
    }

    await this.fetchMe();
  }

  async fetchMe(): Promise<void> {
    this.meStage.loading();

    try {
      const response = await api.get<MeResponse>('/me');

      this.applyMeResponse(response);
      this.meStage.success();
    } catch {
      this.meStage.error();
    }
  }

  async createUser(profile: UserProfile): Promise<void> {
    this.submitStage.loading();

    try {
      const { user } = await api.post<{ user: MeUser }>('/users', {
        name: profile.name.trim(),
        age: profile.age,
        gender: profile.gender,
      });

      this.applyMeResponse({ exists: true, user });
      this.submitStage.success();
    } catch {
      this.submitStage.error();
    }
  }

  async resetProgress(role: Role): Promise<void> {
    this.resetStage.loading();

    try {
      const progress = await api.post<RoleProgress>('/progress/reset', {
        role,
      });

      if (role === 'buyer') {
        this.buyer = cloneProgress(progress);
      } else {
        this.seller = cloneProgress(progress);
      }

      this.resetStage.success();
    } catch {
      this.resetStage.error();
    }
  }
}

export default UserStore;
