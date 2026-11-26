package trades

import "regexp"

// gradedCondition matches the label of a graded slab as the Cadastro writes it in
// Condition: the grader followed by the grade ("PSA 10", "BGS 9.5", "CGC 10").
var gradedCondition = regexp.MustCompile(`(?i)^\s*(PSA|BGS|CGC|SGC|TAG)\s*\d`)

// IsGraded reports whether a single is a graded slab. A slab is worth far more
// than the raw card, so the raw-card market price (TCGplayer or Liga) never
// prices it: its value is the manual one.
func IsGraded(condition string) bool {
	return gradedCondition.MatchString(condition)
}
