import ExamStore from '../ExamStore';
import ResultsStore from '../ResultsStore';
import TrainingStore from '../TrainingStore';
import UserStore from '../UserStore';

class RootStore {
  user: UserStore;
  training: TrainingStore;
  exam: ExamStore;
  results: ResultsStore;

  constructor() {
    this.user = new UserStore();
    this.training = new TrainingStore();
    this.exam = new ExamStore();
    this.results = new ResultsStore();
  }
}

export const rootStore = new RootStore();
