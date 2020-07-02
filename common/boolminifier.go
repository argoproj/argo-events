package common

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/Knetic/govaluate"
	"github.com/pkg/errors"
)

const (
	variableFlag = "VARIABLE-"
)

// Minifier is a bool expression minifier
type Minifier interface {
	GetExpression() string
}

type term []int

type expr struct {
	expression *govaluate.EvaluableExpression
	variables  []string
	minterms   []term
	table      []tableRow
}

type tableRow struct {
	term    term
	postfix bool
}

// NewBoolExpression returns a Minifier instance
func NewBoolExpression(expression string) (Minifier, error) {
	// e.g. (a || b || c) && (a && b)
	expression = strings.ReplaceAll(expression, "-", "\\-")
	ex, err := govaluate.NewEvaluableExpression(expression)
	if err != nil {
		return nil, err
	}
	// Validate
	for _, token := range ex.Tokens() {
		switch token.Kind {
		case govaluate.VARIABLE:
		case govaluate.LOGICALOP:
		case govaluate.CLAUSE:
		case govaluate.CLAUSE_CLOSE:
			continue
		default:
			return nil, errors.New("unsupported symbol found")
		}
	}

	vars := []string{}
	keys := make(map[string]bool)
	for _, v := range ex.Vars() {
		if _, ok := keys[v]; !ok {
			keys[v] = true
			vars = append(vars, v)
		}
	}
	sort.Strings(vars)
	return &expr{
		expression: ex,
		variables:  vars,
	}, nil
}

func (e *expr) GetExpression() string {
	e.generateTable()
	if len(e.variables) == 0 || len(e.table) == 0 {
		return ""
	}
	for _, tr := range e.table {
		if tr.postfix {
			e.minterms = append(e.minterms, tr.term)
		}
	}

	saver := []term{}
	for i := 0; i < len(e.variables) && len(e.minterms) > 1; i++ {
		compared := make([]bool, len(e.minterms))
		for j := 0; j < len(e.minterms)-1; j++ {
			for k := j + 1; k < len(e.minterms); k++ {
				temp := term{}
				for l := 0; l < len(e.variables); l++ {
					if e.minterms[j][l] == e.minterms[k][l] {
						temp = append(temp, l)
					}
				}
				if len(temp) == len(e.variables)-1 {
					saver, compared = e.saveValue(temp, saver, compared, j, k)
				}
			}
		}
		saver = e.addOther(saver, compared)

		if len(saver) > 0 {
			e.minterms = []term{}
			e.minterms = append(e.minterms, saver...)
		}

		// remove duplicates
		for i := 0; i < len(e.minterms); i++ {
			for j := i + 1; j < len(e.minterms); j++ {
				if termEqual(e.minterms[i], e.minterms[j]) {
					// delete j
					e.minterms = append(e.minterms[:j], e.minterms[j+1:]...)
					j--
				}
			}
		}
		saver = []term{}
	}
	return e.getMinified()
}

func (e *expr) saveValue(temp term, saver []term, compared []bool, start, end int) ([]term, []bool) {
	if len(temp) == len(e.variables)-1 {
		for i := 0; i < len(e.minterms[start]); i++ {
			if i == len(temp) {
				temp = append(temp, -1)
			} else if i != temp[i] {
				// insert temp[i] = -1
				temp = append(temp, 0)
				copy(temp[i+1:], temp[i:])
				temp[i] = -1
			}
		}

		t := term{}
		for i := 0; i < len(temp); i++ {
			if temp[i] == -1 {
				t = append(t, -1)
			} else {
				t = append(t, e.minterms[start][i])
			}
		}
		saver = append(saver, t)

		compared[start] = true
		compared[end] = true
	}
	return saver, compared
}

func (e *expr) addOther(saver []term, compared []bool) []term {
	for i := 0; i < len(e.minterms); i++ {
		if len(compared) > 0 && !compared[i] {
			t := term{}
			for j := 0; j < len(e.minterms[i]); j++ {
				t = append(t, e.minterms[i][j])
			}
			saver = append(saver, t)
		}
	}
	return saver
}

func (e *expr) getMinified() string {
	orVars := []string{}
	for _, t := range e.minterms {
		andVars := []string{}
		for i := 0; i < len(t); i++ {
			if t[i] == -1 {
				continue
			}
			andVars = append(andVars, e.variables[i])
		}
		if len(andVars) > 1 {
			orVars = append(orVars, fmt.Sprintf("(%s)", strings.Join(andVars, " && ")))
		} else if len(andVars) == 1 {
			orVars = append(orVars, andVars[0])
		}
	}
	switch {
	case len(orVars) == 1:
		if strings.HasPrefix(orVars[0], "(") && strings.HasSuffix(orVars[0], ")") {
			return orVars[0][1 : len(orVars[0])-1]
		}
		return orVars[0]
	case len(orVars) > 1:
		return strings.Join(orVars, " || ")
	default:
		return ""
	}
}

func (e *expr) generateTable() {
	valueTable := getTable(len(e.variables))
	postfix := e.infixToPostfix()
	for _, t := range valueTable {
		e.table = append(e.table, tableRow{
			term:    t,
			postfix: e.evaluatePostfix(e.variables, t, postfix),
		})
	}
}

func (e *expr) infixToPostfix() []string {
	postfix := []string{}
	operators := stringStack{}
	for _, token := range e.expression.Tokens() {
		switch token.Kind {
		case govaluate.CLAUSE:
			operators.push("(")
		case govaluate.CLAUSE_CLOSE:
			for operators.peek() != "(" {
				postfix = append(postfix, operators.pop())
			}
			// pop up "("
			operators.pop()
		case govaluate.LOGICALOP:
			for !operators.isEmpty() && rank(token.Value) <= rank(operators.peek()) {
				postfix = append(postfix, operators.pop())
			}
			operators.push(fmt.Sprintf("%v", token.Value))
		default:
			// VARIABLE
			postfix = append(postfix, fmt.Sprintf("%s%v", variableFlag, token.Value))
		}
	}

	for !operators.isEmpty() {
		postfix = append(postfix, operators.pop())
	}
	return postfix
}

func (e *expr) evaluatePostfix(vars []string, set term, postfix []string) bool {
	varMap := make(map[string]int)
	for index, v := range e.variables {
		varMap[v] = index
	}

	operands := []int{}
	for _, p := range postfix {
		if strings.HasPrefix(p, variableFlag) {
			v := p[len(variableFlag):]
			index := varMap[v]
			operands = append(operands, set[index])
			continue
		}
		switch {
		case p == "||":
			n := len(operands) - 1
			operands[n-1] = operands[n] + operands[n-1] - operands[n]*operands[n-1]
			operands = operands[:n]
		case p == "&&":
			n := len(operands) - 1
			operands[n-1] = operands[n] * operands[n-1]
			operands = operands[:n]
		}
	}
	return operands[len(operands)-1] > 0
}

type stringStack struct {
	strings []string
}

func (ss *stringStack) push(str string) {
	ss.strings = append(ss.strings, str)
}

func (ss *stringStack) pop() string {
	n := len(ss.strings) - 1
	result := ss.strings[n]
	ss.strings = ss.strings[:n]
	return result
}

func (ss *stringStack) peek() string {
	n := len(ss.strings) - 1
	return ss.strings[n]
}

func (ss *stringStack) isEmpty() bool {
	return len(ss.strings) == 0
}

func getTable(varSize int) []term {
	max := int(math.Pow(float64(2), float64(varSize)))
	result := []term{}
	for i := 0; i < max; i++ {
		result = append(result, getRow(i, varSize))
	}
	return result
}

func getRow(num, varSize int) term {
	binaryStr := strconv.FormatInt(int64(num), 2)
	for len(binaryStr) < varSize {
		binaryStr = "0" + binaryStr
	}
	t := []int{}
	for i := 0; i < len(binaryStr); i++ {
		if binaryStr[i] == '0' {
			t = append(t, 0)
		} else {
			t = append(t, 1)
		}
	}
	return term(t)
}

func rank(operator interface{}) int {
	switch {
	case operator == "||":
		return 1
	case operator == "&&":
		return 2
	default:
		return 0
	}
}

func termEqual(a, b term) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
