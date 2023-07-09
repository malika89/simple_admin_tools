package gorm

import (
	_ "embed"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/zeromicro/go-zero/tools/goctl/api/spec"
	"github.com/zeromicro/go-zero/tools/goctl/util"
	"github.com/zeromicro/go-zero/tools/goctl/util/pathx"
)

const defaultLogicPackage = "logic"

//go:embed template/handler.tpl
var handlerTemplate string

type handlersInfo struct {
	PkgName        string
	ImportPackages string
	Handlers       []handlerInfo
}
type handlerInfo struct {
	HandlerDoc  string
	HandlerName string
	RequestType string
	LogicName   string
	LogicType   string
	Call        string
	HasResp     bool
	HasRequest  bool
	TransErr    bool
}

func genHandler(group spec.Group, route spec.Route, trans bool,newLogicType string) handlerInfo {
	handler := getHandlerName(route)
	handlerPath := getHandlerFolderPath(group, route)
	pkgName := handlerPath[strings.LastIndex(handlerPath, "/")+1:]
	logicName := defaultLogicPackage
	if handlerPath != handlerDir {
		handler = cases.Title(language.English, cases.NoLower).String(handler)
		logicName = pkgName
	}

	// write doc for swagger
	var handlerDoc *strings.Builder
	handlerDoc = &strings.Builder{}

	var isParameterRequest bool
	if route.RequestType != nil && strings.Contains(strings.Join(route.RequestType.Documents(), ""),
		"swagger:parameters") {
		isParameterRequest = true
	} else {
		isParameterRequest = false
	}

	var swaggerPath string
	if strings.Contains(route.Path, ":") {
		swaggerPath = ConvertRoutePathToSwagger(route.Path)
	} else {
		swaggerPath = route.Path
	}

	prefix := group.GetAnnotation(spec.RoutePrefixKey)
	prefix = strings.ReplaceAll(prefix, `"`, "")
	prefix = strings.TrimSpace(prefix)
	if len(prefix) > 0 {
		prefix = path.Join("/", prefix)
	}
	swaggerPath = path.Join("/", prefix, swaggerPath)

	handlerDoc.WriteString(fmt.Sprintf("// swagger:route %s %s %s %s \n", route.Method, swaggerPath,
		group.GetAnnotation("group"), strings.TrimSuffix(handler, "Handler")))
	handlerDoc.WriteString("//\n")
	handlerDoc.WriteString(fmt.Sprintf("%s\n", strings.Join(route.HandlerDoc, " ")))
	handlerDoc.WriteString("//\n")
	handlerDoc.WriteString(fmt.Sprintf("%s\n", strings.Join(route.HandlerDoc, " ")))
	handlerDoc.WriteString("//\n")

	// HasRequest
	if len(route.RequestTypeName()) > 0 && !isParameterRequest {
		handlerDoc.WriteString(fmt.Sprintf(`// Parameters:
			//  + name: body
			//    require: true
			//    in: %s
			//    type: %s
			//
			`, "body", route.RequestTypeName()))
	}
	// HasResp
	if len(route.ResponseTypeName()) > 0 {
		handlerDoc.WriteString(fmt.Sprintf(`// Responses:
			//  200: %s`, route.ResponseTypeName()))
	}

	return handlerInfo{
		HandlerDoc:  handlerDoc.String(),
		HandlerName: handler,
		RequestType: util.Title(route.RequestTypeName()),
		LogicName:   logicName,
		LogicType:   newLogicType,
		Call:        cases.Title(language.English, cases.NoLower).String(strings.TrimSuffix(handler, "Handler")),
		HasResp:     len(route.ResponseTypeName()) > 0,
		HasRequest:  len(route.RequestTypeName()) > 0,
		TransErr:    trans,
	}
}

func doGenToFile(dir, filename string, group spec.Group, data handlersInfo) error {
	return genFile(fileGenConfig{
		dir:             dir,
		subdir:          getHandlerPath(group),
		filename:        strings.TrimSuffix(filename, ".api") + ".go",
		templateName:    "handlerTemplate",
		category:        category,
		templateFile:    handlerTemplateFile,
		builtinTemplate: handlerTemplate,
		data:            data,
	})
}

func genHandlers(dir, rootPkg, fileName string, api *spec.ApiSpec, g *GenGormLogicContextByAPI) error {
	outputDir := filepath.Dir(dir)
	logicWithoutline :=strings.ReplaceAll(strings.ReplaceAll(fileName,"_",""),"-","")
	NewlogicType:=cases.Title(language.English, cases.NoLower).String(strings.TrimSuffix(logicWithoutline, ".api"))
	for _, group := range api.Service.Groups {
		handlerPath := getHandlerPath(group)
		pkgName := handlerPath[strings.LastIndex(handlerPath, "/")+1:]
		handlers := handlersInfo{
			PkgName:        pkgName,
			ImportPackages: genImportPackages(group, rootPkg),
		}
		for _, route := range group.Routes {
			tmpData := genHandler(group, route, g.TransErr, NewlogicType)
			handlers.Handlers = append(handlers.Handlers, tmpData)
		}
		if err := doGenToFile(outputDir, fileName, group, handlers); err != nil {
			fmt.Printf("generate file %s error:%v", fileName, err)
		}
	}

	return nil
}

func genImportPackages(group spec.Group, parentPkg string) string {
	imports := []string{
		fmt.Sprintf("\"%s\"", pathx.JoinPackages(parentPkg, getLogicPath(group))),
		fmt.Sprintf("\"%s\"", pathx.JoinPackages(parentPkg, contextDir)),
	}
	imports = append(imports, fmt.Sprintf("\"%s\"\n", pathx.JoinPackages(parentPkg, typesDir)))
	return strings.Join(imports, "\n\t")
}

func getHandlerBaseName(route spec.Route) (string, error) {
	handler := route.Handler
	handler = strings.TrimSpace(handler)
	handler = strings.TrimSuffix(handler, "handler")
	handler = strings.TrimSuffix(handler, "Handler")

	return handler, nil
}

func getHandlerFolderPath(group spec.Group, route spec.Route) string {
	folder := route.GetAnnotation(groupProperty)
	if len(folder) == 0 {
		folder = group.GetAnnotation(groupProperty)
		if len(folder) == 0 {
			return handlerDir
		}
	}

	folder = strings.TrimPrefix(folder, "/")
	folder = strings.TrimSuffix(folder, "/")

	return path.Join(handlerDir, folder)
}

func getHandlerPath(group spec.Group) string {
	folder := group.GetAnnotation(groupProperty)
	if len(folder) == 0 {
		return handlerDir
	}
	folder = strings.TrimPrefix(folder, "/")
	folder = strings.TrimSuffix(folder, "/")

	return path.Join(handlerDir, folder)
}

func getHandlerName(route spec.Route) string {
	handler, err := getHandlerBaseName(route)
	if err != nil {
		panic(err)
	}

	return handler + "Handler"
}

func getLogicName(route spec.Route) string {
	handler, err := getHandlerBaseName(route)
	if err != nil {
		panic(err)
	}

	return handler
}
