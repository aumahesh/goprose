package templates

import (
	"fmt"
	"os"
	"path"
	"text/template"

	log "github.com/sirupsen/logrus"

	"github.com/aumahesh/goprose/internal/intermediate"
	"github.com/aumahesh/goprose/internal/util"
)

type ftype int

const (
	goModFile ftype = iota
	stateProtoFile
	interfaceFile
	implementationFile
	mainFile
	makeFile
)

var (
	templateFiles = map[ftype]string{
		goModFile:          "go_mod_template.tmpl",
		stateProtoFile:     "state_proto.tmpl",
		implementationFile: "impl_template_go.tmpl",
		interfaceFile:      "interface_template_go.tmpl",
		mainFile:           "main_template_go.tmpl",
		makeFile:           "makefile.tmpl",
	}
)

type TemplateManager struct {
	templatePath string
	outputPath   string
	modulePath   string
	intermediate *intermediate.Program
}

func NewTemplateManager(templatePath string, outputPath string, program *intermediate.Program) (*TemplateManager, error) {
	t := &TemplateManager{
		templatePath: templatePath,
		outputPath:   outputPath,
		modulePath:   fmt.Sprintf("%s/%s", outputPath, program.ModuleName),
		intermediate: program,
	}

	return t, nil
}

func (t *TemplateManager) getFileName(fileType ftype) string {
	switch fileType {
	case goModFile:
		return fmt.Sprintf("%s/go.mod", t.modulePath)
	case stateProtoFile:
		return fmt.Sprintf("%s/proto/state.proto", t.modulePath)
	case interfaceFile:
		return fmt.Sprintf("%s/%s/%s.go", t.modulePath, t.intermediate.PackageName, t.intermediate.InterfaceName)
	case implementationFile:
		return fmt.Sprintf("%s/%s/%s.go", t.modulePath, t.intermediate.PackageName, t.intermediate.ImplementationName)
	case mainFile:
		return fmt.Sprintf("%s/cmd/main.go", t.modulePath)
	case makeFile:
		return fmt.Sprintf("%s/Makefile", t.modulePath)
	}
	return ""
}

func (t *TemplateManager) Render() error {
	os.MkdirAll(fmt.Sprintf("%s/proto", t.modulePath), 0777)
	os.MkdirAll(fmt.Sprintf("%s/%s", t.modulePath, t.intermediate.PackageName), 0777)
	os.MkdirAll(fmt.Sprintf("%s/cmd", t.modulePath), 0777)

	tplFuncs := template.FuncMap{
		"getType": func(key string) string {
			t, err := t.intermediate.GetType(key)
			if err != nil {
				return "int"
			}
			return t
		},
		"isString": func(key string) bool {
			return t.intermediate.IsType(key, "string")
		},
		"getConstantType": func(key string) string {
			t, err := t.intermediate.GetType(key)
			if err != nil {
				return "int"
			}
			return t
		},
		"isConstantString": func(key string) bool {
			return t.intermediate.IsConstantType(key, "string")
		},
		"increment": func(c int) int {
			return c + 1
		},
		"protoName": util.ToCamelCase,
	}

	for fileType, tf := range templateFiles {
		templateFilePath := path.Join(t.templatePath, tf)
		tpl, err := template.New(tf).Funcs(tplFuncs).ParseFiles(templateFilePath)
		if err != nil {
			return err
		}
		outfile := t.getFileName(fileType)
		log.Debugf("Render template %s at %s", tpl.Name(), outfile)
		f, err := os.Create(outfile)
		if err != nil {
			return err
		}
		err = tpl.Execute(f, t.intermediate)
		if err != nil {
			return err
		}
	}
	return nil
}
