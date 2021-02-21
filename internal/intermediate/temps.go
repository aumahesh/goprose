package intermediate

import (
	"fmt"
)

const (
	tempFmt = "temp%d"
)

type TempVariable struct {
	tempName string
	tempType string
}

type TempsManager struct {
	counter   int
	variables map[string]*TempVariable
}

func NewTempsManager() *TempsManager {
	return &TempsManager{
		counter:   0,
		variables: map[string]*TempVariable{},
	}
}

func (tm *TempsManager) NewTempVariable(ttype string) string {
	name := fmt.Sprintf(tempFmt, tm.counter)
	tm.counter++
	t := &TempVariable{
		tempName: name,
		tempType: ttype,
	}
	tm.variables[name] = t
	return name
}

func (tm *TempsManager) GetTempType(name string) (string, error) {
	t, ok := tm.variables[name]
	if !ok {
		return "", fmt.Errorf("temp variable %s not found", name)
	}
	return t.tempType, nil
}
