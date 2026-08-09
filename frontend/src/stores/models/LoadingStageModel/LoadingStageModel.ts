import { actionBound, computed, makeObservable, observable } from 'mobx';

import { MetaState } from './types';

export class LoadingStageModel {
  private _value: MetaState;

  constructor(value: MetaState = MetaState.initial) {
    this._value = value;

    makeObservable<LoadingStageModel, '_value'>(this, {
      _value: observable,

      value: computed,
      isNotStarted: computed,
      isLoading: computed,
      isSuccess: computed,
      isError: computed,
      isFinished: computed,

      loading: actionBound,
      success: actionBound,
      error: actionBound,
      reset: actionBound,
    });
  }

  get value(): MetaState {
    return this._value;
  }

  get isNotStarted(): boolean {
    return this._value === MetaState.initial;
  }

  get isLoading(): boolean {
    return this._value === MetaState.loading;
  }

  get isSuccess(): boolean {
    return this._value === MetaState.success;
  }

  get isError(): boolean {
    return this._value === MetaState.failed;
  }

  get isFinished(): boolean {
    return (
      this._value === MetaState.success || this._value === MetaState.failed
    );
  }

  loading(): void {
    this._value = MetaState.loading;
  }

  success(): void {
    this._value = MetaState.success;
  }

  error(): void {
    this._value = MetaState.failed;
  }

  reset(): void {
    this._value = MetaState.initial;
  }
}
