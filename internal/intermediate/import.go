package intermediate

import (
	"fmt"
	"path"

	log "github.com/sirupsen/logrus"
)

type Import struct {
	PackageId  string
	ImportPath string
}

func NewImports(packages []string) (map[string]*Import, error) {
	importMap := map[string]*Import{}

	defaultImports := map[string]bool{
		"context":                          true,
		"net":                              true,
		"time":                             true,
		"github.com/golang/protobuf/proto": true,
		"github.com/dmichael/go-multicast/multicast": true,
		"github.com/sirupsen/logrus":                 true,
	}

	for _, pkg := range packages {
		base := path.Base(pkg)

		imp := &Import{
			PackageId:  base,
			ImportPath: pkg,
		}

		x, present := defaultImports[pkg]
		if present && x {
			continue
		}

		_, present = importMap[base]
		if present {
			return nil, fmt.Errorf("duplicate import statement: %s", base)
		}

		importMap[base] = imp
		log.Debugf("import: %+v", imp)
	}

	return importMap, nil
}
