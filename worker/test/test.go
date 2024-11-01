package main

import (
	"fmt"
	"math/big"
)

// CalculateTerm calcula un término de π usando la fórmula de BBP.
func CalculateTerm(k uint64) *big.Float {
	precision := uint(1000)
	// Crear un nuevo número de punto flotante con la precisión deseada
	term := new(big.Float).SetPrec(precision)

	// Calcular cada parte de la fórmula de BBP
	part1 := new(big.Float).Quo(big.NewFloat(4), big.NewFloat(float64(8*k+1)))
	part2 := new(big.Float).Quo(big.NewFloat(2), big.NewFloat(float64(8*k+4)))
	part3 := new(big.Float).Quo(big.NewFloat(1), big.NewFloat(float64(8*k+5)))
	part4 := new(big.Float).Quo(big.NewFloat(1), big.NewFloat(float64(8*k+6)))

	// Sumar los términos
	term = term.Add(term, part1)
	term = term.Sub(term, part2)
	term = term.Sub(term, part3)
	term = term.Sub(term, part4)

	// Multiplicar por 1/16^k
	power := new(big.Int).Exp(big.NewInt(16), big.NewInt(int64(k)), nil)
	multiplier := new(big.Float).SetPrec(precision).Quo(big.NewFloat(1), new(big.Float).SetInt(power))

	term.Mul(term, multiplier)

	return term
}

func main() {
	termsToCalculate := 100 // Número de términos que deseas calcular
	pi := new(big.Float).SetPrec(1000).SetFloat64(0)

	for k := uint64(0); k < uint64(termsToCalculate); k++ {
		term := CalculateTerm(k)
		pi.Add(pi, term)
	}

	// Mostrar el resultado final de π
	fmt.Printf("Calculated π: %.600f\n", pi)
}
