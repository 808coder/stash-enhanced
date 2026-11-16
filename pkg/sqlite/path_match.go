package sqlite

import (
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

func pathHasWildcard(path string) bool {
	return strings.Contains(path, "*")
}

// escapeLike escapes LIKE wildcards so that s matches literally with
// ESCAPE '\'.
func escapeLike(s string) string {
	return likeEscaper.Replace(s)
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func pathToLikePattern(path string) string {
	parts := strings.Split(path, "*")
	for i, part := range parts {
		parts[i] = escapeLike(part)
	}

	return strings.Join(parts, "%")
}

func pathLike(col exp.IdentifierExpression, pattern string) exp.Expression {
	return goqu.L(`? LIKE ? ESCAPE '\'`, col, pathToLikePattern(pattern))
}

func pathEqNoCase(col exp.IdentifierExpression, value string) exp.Expression {
	return goqu.L("? = ? COLLATE NOCASE", col, value)
}
