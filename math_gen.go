package main

import "fmt"

type mathGenerator struct {
	ops []mathOp
	max int
}

type mathOp int

const (
	add mathOp = iota
	subtract
	multiply
	divide
	opCount
)

func (op mathOp) String() string {
	switch op {
	case add:
		return "+"
	case subtract:
		return "-"
	case multiply:
		return "*"
	case divide:
		return "/"
	default:
		panic("invalid mathOp")
	}
}

type assignment struct {
	question string
	answer   int
}

// generate creates an equation with two operands.
func (g mathGenerator) generate(rand func() int) assignment {
	op := g.ops[rand()%len(g.ops)]
	var a, b, result int
	switch op {
	case add:
		result = rand() % (g.max + 1)
		if result == 0 {
			a, b = 0, 0
		} else {
			a = rand() % result
			b = result - a
		}
	case subtract:
		result = rand() % (g.max + 1)
		if result == g.max {
			a, b = result, 0
		} else {
			a = result + rand()%(g.max-result)
			b = a - result
		}
	case multiply:
		result = rand() % (g.max + 1)
		if result == 0 {
			a, b = 0, rand()%(g.max+1)
			if rand()%2 == 0 {
				a, b = b, a
			}
		} else {
			a = 1 + rand()%result
			for result%a != 0 {
				a--
			}
			b = result / a
		}
	case divide:
		result = 1 + rand()%(g.max)
		b = 1 + rand()%(g.max)
		for result*b > g.max {
			b--
		}
		a = result * b
	}
	return assignment{
		question: fmt.Sprintf("%d %s %d", a, op, b),
		answer:   result,
	}
}
