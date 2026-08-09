import { makeAutoObservable } from 'mobx';

import { api } from '@/shared/api';

import { LoadingStageModel } from '../models';

export type Role = 'buyer' | 'seller';

export type Gender = 'male' | 'female';

export type RoleProgress = {
  training: {
    currentStep: number;
    totalSteps: number;
  };
  isExamPassed: boolean;
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

export type CreateUserResponse = {
  user: MeUser;
};

export type UserProfile = {
  name: string;
  age: string;
  gender: Gender;
};

export const isRole = (value: unknown): value is Role =>
  value === 'buyer' || value === 'seller';

export const isTrainingPassed = (progress: RoleProgress): boolean =>
  progress.training.currentStep === progress.training.totalSteps;

export const hasRoleProgress = (progress: RoleProgress): boolean =>
  progress.training.currentStep > 0 || progress.isExamPassed;

const emptyProgress = (): RoleProgress => ({
  training: {
    currentStep: 0,
    totalSteps: 0,
  },
  isExamPassed: false,
});

const cloneProgress = (progress: RoleProgress): RoleProgress => ({
  training: { ...progress.training },
  isExamPassed: progress.isExamPassed,
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

    void this.fetchMe();
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

  setTrainingProgress(role: Role, currentStep: number) {
    const progress = this.getProgress(role);

    progress.training.currentStep = currentStep;
  }

  markExamPassed(role: Role) {
    this.getProgress(role).isExamPassed = true;
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

  async fetchMe(): Promise<void> {
    this.meStage.loading();

    try {
      const response = await api.get<MeResponse>('/api/me');

      this.applyMeResponse(response);
      this.meStage.success();
    } catch {
      this.meStage.error();
    }
  }

  async createUser(profile: UserProfile): Promise<void> {
    this.submitStage.loading();

    try {
      const response = await api.post<CreateUserResponse>('/api/users', {
        name: profile.name,
        age: profile.age,
        gender: profile.gender,
      });

      this.applyMeResponse({ exists: true, user: response.user });
      this.submitStage.success();
    } catch {
      this.submitStage.error();
    }
  }

  async resetProgress(role: Role): Promise<void> {
    this.resetStage.loading();

    try {
      const progress = await api.post<RoleProgress>('/api/progress/reset', {
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
