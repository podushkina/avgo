export type Role = 'buyer' | 'seller';
export type Verdict = 'safe' | 'risky' | 'dangerous';
export type Difficulty = 'easy' | 'medium' | 'hard';

export interface User {
  id: string;
  external_id: string;
  created_at: string;
}

export interface Scenario {
  id: number;
  role: Role;
  order_index: number;
  title: string;
  situation: string;
  question: string;
  options: string[];
}

export interface CheckResult {
  is_correct: boolean;
  your_verdict: Verdict;
  your_outcome: string;
  points: number;
  correct_option: number;
  correct_option_text: string;
  correct_outcome: string;
  explanation: string;
  red_flags: string[];
}

export interface Review {
  scenario_id: number;
  title: string;
  question: string;
  answered: boolean;
  is_correct: boolean;
  your_option: number;
  your_option_text: string;
  your_verdict: Verdict;
  your_outcome: string;
  correct_option: number;
  correct_option_text: string;
  correct_outcome: string;
  explanation: string;
  red_flags: string[];
  points: number;
}

export interface AttemptResult {
  attempt_id: string;
  role: Role;
  correct: number;
  total: number;
  percent: number;
  score: number;
  max_score: number;
  level: string;
  perfect: boolean;
  reviews: Review[];
  mistakes: Review[];
  missed_red_flags: string[];
  suggested_difficulty: Difficulty;
  completed_at: string;
}

export interface ProgressEntry {
  id: string;
  role: Role;
  correct_count: number;
  total_count: number;
  percent: number;
  score: number;
  level: string;
  mistakes: Review[];
  completed_at: string;
}

export interface DialogSession {
  session_id: string;
  role: Role;
  difficulty: Difficulty;
  opening_message: string;
  max_turns: number;
}

export interface Finding {
  code: string;
  title: string;
  detail: string;
  quote: string;
}

export interface DialogReport {
  tactics: Finding[];
  mistakes: Finding[];
  turns: number;
  survived: boolean;
  verdict: string;
  advice: string[];
}
