package templates

import (
	"fmt"
	"os"
	"path"
	"text/template"

	log "github.com/sirupsen/logrus"

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
		goModFile:          "resources/go_mod_template.tmpl",
		stateProtoFile:     "resources/state_proto.tmpl",
		implementationFile: "resources/impl_template_go.tmpl",
		interfaceFile:      "resources/interface_template_go.tmpl",
		mainFile:           "resources/main_template_go.tmpl",
		makeFile:           "resources/makefile.tmpl",
	}
)

type ProseProgram struct {
	Org                string
	ModuleName         string
	PackageName        string
	InterfaceName      string
	ImplementationName string
	Variables          map[string]string
	InitialState       map[string]interface{}
}

func (pv *ProseProgram) GetType(key string) (string, error) {
	v, ok := pv.Variables[key]
	if !ok {
		return "", fmt.Errorf("%s not found", key)
	}
	return v, nil
}

func (pv *ProseProgram) IsType(key string, tgt string) bool {
	t, err := pv.GetType(key)
	if err != nil {
		return false
	}
	return t == tgt
}

type TemplateManager struct {
	outputPath string
}

func NewTemplateManager(outputPath string) (*TemplateManager, error) {
	t := &TemplateManager{
		outputPath: outputPath,
	}
	return t, nil
}

func (t *TemplateManager) getFileName(fileType ftype, pv *ProseProgram) string {
	switch fileType {
	case goModFile:
		return fmt.Sprintf("%s/go.mod", t.outputPath)
	case stateProtoFile:
		return fmt.Sprintf("%s/proto/state.proto", t.outputPath)
	case interfaceFile:
		return fmt.Sprintf("%s/%s/%s_intf.go", t.outputPath, pv.PackageName, pv.InterfaceName)
	case implementationFile:
		return fmt.Sprintf("%s/%s/%s_impl.go", t.outputPath, pv.PackageName, pv.ImplementationName)
	case mainFile:
		return fmt.Sprintf("%s/cmd/main.go", t.outputPath)
	case makeFile:
		return fmt.Sprintf("%s/Makefile", t.outputPath)
	}
	return ""
}

func (t *TemplateManager) Render(pv *ProseProgram) error {
	os.MkdirAll(fmt.Sprintf("%s/proto", t.outputPath), 0777)
	os.MkdirAll(fmt.Sprintf("%s/%s", t.outputPath, pv.PackageName), 0777)
	os.MkdirAll(fmt.Sprintf("%s/cmd", t.outputPath), 0777)

	tplFuncs := template.FuncMap{
		"isString": func(key string) bool {
			return pv.IsType(key, "string")
		},
		"increment": func(c int) int {
			return c + 1
		},
		"protoName": util.ToCamelCase,
	}

	for fileType, tf := range templateFiles {
		tpl, err := template.New(path.Base(tf)).Funcs(tplFuncs).ParseFiles(tf)
		if err != nil {
			return err
		}
		outfile := t.getFileName(fileType, pv)
		log.Debugf("Render template %s at %s", tpl.Name(), outfile)
		f, err := os.Create(outfile)
		if err != nil {
			return err
		}
		err = tpl.Execute(f, pv)
		if err != nil {
			return err
		}
	}
	return nil
}
