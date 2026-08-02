package db

import (
	"crypto/rand"
	"fmt"
)

// bankPrefix identifica a este banco en los numeros de cuenta que genera
// (mismo prefijo que usa el dataset de prueba: "4001-XXXX-XXXX-NNNN").
const bankPrefix = "4001"

// GenerateAccountNumber crea un numero de cuenta con formato
// "4001-XXXX-XXXX-NNNN". No garantiza unicidad por si solo (dos llamadas
// podrian coincidir, aunque es extremadamente improbable) - quien lo use
// debe manejar el caso de colision contra la restriccion UNIQUE de la
// base de datos y reintentar si hace falta.
func GenerateAccountNumber() (string, error) {
	block1, err := randomDigits(4)
	if err != nil {
		return "", err
	}
	block2, err := randomDigits(4)
	if err != nil {
		return "", err
	}
	block3, err := randomDigits(4)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s-%s-%s", bankPrefix, block1, block2, block3), nil
}

func randomDigits(n int) (string, error) {
	digits := make([]byte, n)
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("error generando numero aleatorio: %w", err)
	}
	for i, b := range buf {
		digits[i] = '0' + (b % 10)
	}
	return string(digits), nil
}
