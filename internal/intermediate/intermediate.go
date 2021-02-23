package intermediate

import (
	"fmt"
	"strings"

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

	g.intermediateProgram.Constants = map[string]*Variable{}
	g.intermediateProgram.Variables = map[string]*Variable{}
	g.intermediateProgram.ConstantInitFunctions = map[string]*Expression{}
	g.intermediateProgram.VariableInitFunctions = map[string]*Expression{}

	translatorFuncs := []func() error{
		g.doImportDeclarations,
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

func (g *translator) doImportDeclarations() error {
	log.Debugf("Processing imports...")

	importMap, err := NewImports(g.parsedProgram.Packages)
	if err != nil {
		return err
	}

	g.intermediateProgram.Imports = importMap

	return nil
}

func (g *translator) doConstantDeclarations() error {
	log.Debugf("Processing constants...")

	sensorName := StringValue(g.parsedProgram.Sensor)

	for _, constDefinitions := range g.parsedProgram.ConstDeclarations {
		if len(constDefinitions.Ids) != len(constDefinitions.Values) && len(constDefinitions.Values) != 0 {
			return fmt.Errorf("Error: constant declaration at %s invalid: default value assignment has %d on LHS, and %d on RHS",
				constDefinitions.Pos, len(constDefinitions.Ids), len(constDefinitions.Values))
		}
		for index, id := range constDefinitions.Ids {
			idStr := StringValue(id.Id)
			_, declared := g.intermediateProgram.Constants[idStr]
			if declared {
				return fmt.Errorf("Error: %s variable is already declared: redeclaration @ %s",
					idStr, id.Pos)
			}
			if id.Source != nil {
				return fmt.Errorf("Error: constant cannot point to another sensor/process: %s @ %s",
					idStr, id.Source.Pos)
			}
			v, err := NewVariable(idStr, "", StringValue(constDefinitions.VariableType))
			if err != nil {
				return err
			}
			g.intermediateProgram.Constants[idStr] = v
			log.Debugf("Constant %s: %+v", idStr, v)

			var defaultValue *parser.Expr
			if constDefinitions.Values != nil {
				defaultValue = constDefinitions.Values[index]
				initExpr, err := NewExpression(defaultValue,
					sensorName,
					g.intermediateProgram.Constants,
					g.intermediateProgram.Variables,
					g.tempsManager)
				if err != nil {
					return fmt.Errorf("Error: error in constructing init function for %s @ %s: %s",
						id.Id, defaultValue.Pos, err)
				}
				g.intermediateProgram.ConstantInitFunctions[idStr] = initExpr
			}
		}
	}

	return nil
}

func (g *translator) doVariableDeclarations() error {
	log.Debugf("Processing variables...")

	sensorName := StringValue(g.parsedProgram.Sensor)

	for _, varDefinitions := range g.parsedProgram.VariableDeclarations {
		if len(varDefinitions.Ids) != len(varDefinitions.Values) && len(varDefinitions.Values) != 0 {
			return fmt.Errorf("Error: variable declaration at %s invalid: default value assignment has %d on LHS, and %d on RHS",
				varDefinitions.Pos, len(varDefinitions.Ids), len(varDefinitions.Values))
		}
		for index, id := range varDefinitions.Ids {
			idStr := StringValue(id.Id)
			_, declared := g.intermediateProgram.Variables[idStr]
			if declared {
				return fmt.Errorf("Error: %s variable is already declared", idStr)
			}
			if id.Source != nil {
				if id.Source.VariableId != nil {
					return fmt.Errorf("Error: cannot define a variable that points to another sensor/process: %s @ %s",
						idStr, id.Source.Pos)
				}
				if id.Source.Source != nil && StringValue(id.Source.Source) != sensorName {
					return fmt.Errorf("Error: cannot define a variable that points to another sensor/process: %s @ %s",
						idStr, id.Source.Pos)
				}
			}
			v, err := NewVariable(idStr, StringValue(varDefinitions.Access), StringValue(varDefinitions.VariableType))
			if err != nil {
				return fmt.Errorf("Error: processing variable declaraiton failed: %s", id.Pos)
			}
			g.intermediateProgram.Variables[idStr] = v
			log.Debugf("Variable %s: %+v", idStr, v)

			var defaultValue *parser.Expr
			if varDefinitions.Values != nil {
				defaultValue = varDefinitions.Values[index]
				initExpr, err := NewExpression(defaultValue,
					sensorName,
					g.intermediateProgram.Constants,
					g.intermediateProgram.Variables,
					g.tempsManager)
				if err != nil {
					return fmt.Errorf("Error: error in constructing init function for %s @ %s: %s",
						id.Id, defaultValue.Pos, err)
				}
				g.intermediateProgram.VariableInitFunctions[idStr] = initExpr
			}

		}
	}

	return nil
}

func (g *translator) doGuardedStatements() error {
	log.Debugf("Processing statements...")

	sensorName := StringValue(g.parsedProgram.Sensor)

	g.intermediateProgram.Statements = map[string]*GuardedStatement{}

	for index, stmt := range g.parsedProgram.Statements {
		statementFunc := fmt.Sprintf("doAction%d", index)
		s, err := NewStatement(stmt, sensorName, g.intermediateProgram.Constants, g.intermediateProgram.Variables, g.tempsManager)
		if err != nil {
			return fmt.Errorf("Error: statement %s errored: %s", stmt.Pos, err)
		}
		g.intermediateProgram.Statements[statementFunc] = s

		log.Debugf("%s", strings.Join(s.Code, "\n"))
	}
	return nil
}
