package gorm

import (
	_ "embed"
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/zeromicro/go-zero/tools/goctl/api/spec"
	"github.com/zeromicro/go-zero/tools/goctl/config"
	"github.com/zeromicro/go-zero/tools/goctl/util/format"
	"github.com/zeromicro/go-zero/tools/goctl/util/pathx"
	"github.com/zeromicro/go-zero/tools/goctl/vars"
)

const contextFilename = "service_context"

//go:embed template/svc.tpl
var contextTemplate string

func getMiddleWares(rootPkg string, api *spec.ApiSpec) (string, string, string) {
	var (
		middlewareStr        string
		middlewareAssignment string
		configImport         string
	)
	middlewares := getMiddleware(api)
	for _, item := range middlewares {
		middlewareStr += fmt.Sprintf("%s rest.Middleware\n", item)
		name := strings.TrimSuffix(item, "Middleware") + "Middleware"
		middlewareAssignment += fmt.Sprintf("%s: %s,\n", item,
			fmt.Sprintf("middleware.New%s().%s", cases.Title(language.English, cases.NoLower).String(name), "Handle"))
	}
	if len(middlewareStr) > 0 {
		configImport += "\n\t\"" + pathx.JoinPackages(rootPkg, middlewareDir) + "\""
		configImport += fmt.Sprintf("\n\t\"%s/rest\"", vars.ProjectOpenSourceURL)
	}
	return middlewareStr, middlewareAssignment, configImport

}
func genServiceContext(dir, rootPkg string, cfg *config.Config, g *GenGormLogicContextByAPI, extraMap map[string]string) error {
	filename, err := format.FileNamingFormat(cfg.NamingFormat, contextFilename)
	if err != nil {
		return err
	}
	var (
		rpcNewPbClientStr string
	)
	configImport := "\"" + pathx.JoinPackages(rootPkg, configDir) + "\""
	if g.UseI18n {
		configImport += fmt.Sprintf("\n\ti18n2 \"%s/internal/i18n\"", rootPkg)
	}

	if g.UseCasbin {
		configImport += fmt.Sprintf("\n\t\"%s/internal/middleware\"", rootPkg)
	}
	array := strings.Split(g.RpcPackageName, ".")
	if len(array) == 2 {
		rpcNewPbClientStr = fmt.Sprintf("%s.New%s", array[0], array[1])
	} else {
		rpcNewPbClientStr = g.RpcPackageName
	}
	outPutDir := filepath.Dir(dir)
	return genFile(fileGenConfig{
		dir:             outPutDir,
		subdir:          contextDir,
		filename:        strings.TrimSuffix(filename,".api") + ".go",
		templateName:    "contextTemplate",
		category:        category,
		templateFile:    contextTemplateFile,
		builtinTemplate: contextTemplate,
		data: map[string]any{
			"configImport":         configImport + extraMap["configImport"],
			"config":               "config.Config",
			"useCasbin":            g.UseCasbin,
			"useI18n":              g.UseI18n,
			"projectPackage":       rootPkg,
			"useTrans":             g.TransErr,
			"rpcName":              g.RpcName,
			"rpcPbClient":          g.RpcPackageName,
			"rpcNewPbClient":       rpcNewPbClientStr,
			"middleware":           extraMap["middlewareStr"],
			"middlewareAssignment": extraMap["middlewareAssignment"]},
	})
}
