package gorm

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/zeromicro/go-zero/tools/goctl/api/spec"
	apiutil "github.com/zeromicro/go-zero/tools/goctl/api/util"
	"github.com/zeromicro/go-zero/tools/goctl/config"
	"github.com/zeromicro/go-zero/tools/goctl/util"
	"github.com/zeromicro/go-zero/tools/goctl/util/format"
)

const typesFile = "types"

//go:embed template/types.tpl
var typesTemplate string

// BuildTypes gen types to string
func BuildTypes(types []spec.Type, config *config.Config) (string, error) {
	var builder strings.Builder
	first := true
	for _, tp := range types {
		if first {
			first = false
		} else {
			builder.WriteString("\n\n")
		}
		if err := writeType(&builder, tp, config); err != nil {
			return "", apiutil.WrapErr(err, "Type "+tp.Name()+" generate error")
		}
	}

	return builder.String(), nil
}
func genTypes(cfg *config.Config, api *spec.ApiSpec) (string, error) {
	val, err := BuildTypes(api.Types, cfg)
	if err != nil {
		return "", err
	}
	return val, err
}

func genTypesFile(dir string, cfg *config.Config, typesVal string) error {
	outPutDir := filepath.Dir(dir)
	typeFilename, err := format.FileNamingFormat(cfg.NamingFormat, typesFile)
	if err != nil {
		return err
	}

	typeFilename = strings.TrimSuffix(typeFilename,".api") + ".go"
	filename := path.Join(outPutDir, typesDir, typeFilename)
	os.Remove(filename)

	return genFile(fileGenConfig{
		dir:             outPutDir,
		subdir:          typesDir,
		filename:        typeFilename,
		templateName:    "typesTemplate",
		category:        category,
		templateFile:    typesTemplateFile,
		builtinTemplate: typesTemplate,
		data: map[string]any{
			"types":        typesVal,
			"containsTime": false,
		},
	})
}

func writeType(writer io.Writer, tp spec.Type, config *config.Config) error {
	structType, ok := tp.(spec.DefineStruct)
	if !ok {
		return fmt.Errorf("unspport struct type: %s", tp.Name())
	}

	// write doc for swagger
	swaggerStr := strings.Builder{}
	for _, v := range structType.Documents() {
		fmt.Fprintf(writer, "%s\n", v)
		swaggerStr.WriteString(v)
	}

	if !strings.Contains(swaggerStr.String(), "swagger") {
		if strings.HasSuffix(tp.Name(), "Req") || strings.HasSuffix(tp.Name(), "Info") {
			fmt.Fprintf(writer, "// swagger:model %s \n", tp.Name())
		} else if strings.HasSuffix(tp.Name(), "Resp") {
			fmt.Fprintf(writer, "// swagger:model %s \n", tp.Name())
		}
	}

	fmt.Fprintf(writer, "type %s struct {\n", util.Title(tp.Name()))
	for _, member := range structType.Members {
		if member.IsInline {
			if _, err := fmt.Fprintf(writer, "%s\n", cases.Title(language.English, cases.NoLower).String(member.Type.Name())); err != nil {
				return err
			}

			continue
		}

		if err := writeProperty(writer, member.Name, member.Tag, member.GetComment(), member.Type, member.Docs, 1); err != nil {
			return err
		}
	}
	fmt.Fprintf(writer, "}")
	return nil
}
