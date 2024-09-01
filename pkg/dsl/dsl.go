//
// TODO This is a placeholder package for future Shui-defined expressions not provided by the
// expression language. Functions here in its current state are just examples and for testing.

package dsl

import (
	"fmt"
	_ "log/slog"

	"github.com/expr-lang/expr"
)

func addOneToNumber(i int) int {
	return i + 1
}

func addOneToStr(i string) string {
	return i + "1"
}

func Expr(result string) interface{} {
	env := map[string]interface{}{
		"addOneToNumber": addOneToNumber,
		"addOneToStr":    addOneToStr,
	}

	code := fmt.Sprintf("addOneToStr(\"%s\")", result)

	program, err := expr.Compile(code, expr.Env(env))
	if err != nil {
		panic(err)
	}

	output, err := expr.Run(program, env)
	if err != nil {
		panic(err)
	}

	return output
}
