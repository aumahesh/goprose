package intermediate

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/aumahesh/goprose/internal/parser"
)

const (
	defaultOrg = "aumahesh.com/prose"
)

type translator struct {
	parsedProgram       *parser.Program
	intermediateProgram *Program
	tempsManager        *TempsManager
}

func (g *translator) do() error {
	g.intermediateProgram.Org = defaultOrg
	g.intermediateProgram.ModuleName = StringValue(g.parsedProgram.Name)
	g.intermediateProgram.PackageName = "internal"
	g.intermediateProgram.InterfaceName = fmt.Sprintf("%s_intf", StringValue(g.parsedProgram.Name))
	g.intermediateProgram.ImplementationName = fmt.Sprintf("%s_impl", StringValue(g.parsedProgram.Name))

	translatorFuncs := []func() error{
		g.doConstantDeclarations,
		g.doVariableDeclarations,
		g.doGuardedStatements,
	}

	for _, translatorFunc := range translatorFuncs {
		err := translatorFunc()
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *translator) doConstantDeclarations() error {
	g.intermediateProgram.Constants = map[string]*Variable{}

	for _, constDefinitions := range g.parsedProgram.ConstDeclarations {
		for _, id := range constDefinitions.Ids {
			idStr := StringValue(id.Id)
			_, declared := g.intermediateProgram.Constants[idStr]
			if declared {
				return fmt.Errorf("Error: %s variable is already declared", idStr)
			}
			if id.Source != nil {
				return fmt.Errorf("Error: constant cannot point to another sensor/process: %s", idStr)
			}
			v, err := NewVariable(idStr, "", StringValue(constDefinitions.VariableType), nil)
			if err != nil {
				return err
			}
			g.intermediateProgram.Constants[idStr] = v
			log.Debugf("Constant %s: %+v", idStr, v)
		}
	}

	return nil
}

func (g *translator) doVariableDeclarations() error {
	sensorName := StringValue(g.parsedProgram.Sensor)

	g.intermediateProgram.Variables = map[string]*Variable{}

	for _, varDefinitions := range g.parsedProgram.VariableDeclarations {
		for _, id := range varDefinitions.Ids {
			idStr := StringValue(id.Id)
			_, declared := g.intermediateProgram.Variables[idStr]
			if declared {
				return fmt.Errorf("Error: %s variable is already declared", idStr)
			}
			if id.Source != nil {
				if id.Source.VariableId != nil {
					return fmt.Errorf("Error: cannot define a variable that points to another sensor/process: %s", idStr)
				}
				if id.Source.Source != nil && StringValue(id.Source.Source) != sensorName {
					return fmt.Errorf("Error: cannot define a variable that points to another sensor/process: %s", idStr)
				}
			}
			v, err := NewVariable(idStr, StringValue(varDefinitions.Access), StringValue(varDefinitions.VariableType), nil)
			if err != nil {
				return err
			}
			g.intermediateProgram.Variables[idStr] = v
			log.Debugf("Variable %s: %+v", idStr, v)
		}
	}

	return nil
}

func (g *translator) doGuardedStatements() error {
	g.intermediateProgram.Statements = map[string]*GuardedStatement{}

	for index, stmt := range g.parsedProgram.Statements {
		log.Debugf("Processing statement: %d, %+v", index, stmt)
		statementFunc := fmt.Sprintf("doAction%d", index)
		s, err := NewStatement()
		if err != nil {
			return err
		}
		g.intermediateProgram.Statements[statementFunc] = s
	}
	return nil
}
