// A graded slab carries the grader and the grade in its condition, as the
// Cadastro writes it: "PSA 10", "BGS 9.5", "CGC 10", "SGC 10", "TAG 10".
const GRADED_CONDITION = /^\s*(PSA|BGS|CGC|SGC|TAG)\s*\d/i;

export function isGradedCondition(condition: string): boolean {
  return GRADED_CONDITION.test(condition);
}

/** The grading company of a slab ("PSA"), from its grade ("PSA 10"). */
export function graderOf(grade: string): string {
  return grade.trim().split(/\s+|(?=\d)/)[0].toUpperCase();
}
