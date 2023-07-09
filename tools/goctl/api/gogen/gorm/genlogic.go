package gorm

import (
	_ "embed"
	"fmt"
	"github.com/gookit/color"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/zeromicro/go-zero/tools/goctl/api/parser/g4/gen/api"
	"github.com/zeromicro/go-zero/tools/goctl/api/spec"
	"github.com/zeromicro/go-zero/tools/goctl/config"
	"github.com/zeromicro/go-zero/tools/goctl/util/format"
	"github.com/zeromicro/go-zero/tools/goctl/util/pathx"
	"github.com/zeromicro/go-zero/tools/goctl/vars"
)

//go:embed template/logic.tpl
var logicTemplate string

type logicsInfo struct {
	PkgName         string
	ImportPackages  string
	Logic           string
	useUUID         bool
	useI18n         bool
	rpcPackageLower string
	rpcName         string
	Logics          []logicInfo
}
type logicInfo struct {
	Method       string
	Function     string
	ResponseType string
	ReturnString string
	Request      string
	SetLogic     string
}

func genLogics(dir, rootPkg, fileName string, cfg *config.Config, api *spec.ApiSpec, ctx *GenGormLogicContextByAPI) error {
	color.Green.Printf("***** gen logic file for :%s\n", fileName)
	for _, g := range api.Service.Groups {
		var (
			imports         string
			method          string
			rpcPackageLower string
		)
		handlerPath := getHandlerPath(g)
		pkgName := handlerPath[strings.LastIndex(handlerPath, "/")+1:]
		array := strings.Split(ctx.RpcPackageName, ".")
		if len(array) == 2 {
			rpcPackageLower = strings.ToLower(array[1])
		} else {
			rpcPackageLower = strings.ToLower(ctx.RpcPackageName)
		}
		logicWithoutline :=strings.ReplaceAll(strings.ReplaceAll(fileName,"_",""),"-","")
		logics := logicsInfo{
			PkgName:         pkgName,
			Logic:           cases.Title(language.English, cases.NoLower).String(strings.TrimSuffix(logicWithoutline, ".api")),
			useI18n:         ctx.UseI18n,
			useUUID:         false,
			rpcName:         ctx.RpcName,
			rpcPackageLower: rpcPackageLower,
		}
		for _, r := range g.Routes {
			logic_, imports_, responseString_, returnString_, requestString_, err := genLogicByRoute(rootPkg, r)
			if err != nil {
				return err
			}
			logicLower := strings.ToLower(logic_)
			if strings.Contains(logicLower, "create") || strings.Contains(logicLower, "add") {
				method = "Create"
			} else if strings.Contains(logicLower, "del") || strings.Contains(logicLower, "remove") {
				method = "Delete"
			} else if strings.Contains(logicLower, "update") || strings.Contains(logicLower, "edit") {
				method = "Update"
			} else if strings.Contains(logicLower, "get") && strings.Contains(logicLower, "byid") {
				method = "GetById"
			} else if strings.Contains(logicLower, "get") && strings.Contains(logicLower, "list") {
				method = "GetList"
			} else {
				method = "custom"
			}
			logicSingleRoute := logicInfo{
				Method:       method,
				Function:     cases.Title(language.English, cases.NoLower).String(strings.TrimSuffix(logic_, "Logic")),
				ResponseType: responseString_,
				ReturnString: returnString_,
				Request:      requestString_,
				SetLogic:     genExtraLogic(api, requestString_, method),
			}
			logics.Logics = append(logics.Logics, logicSingleRoute)
			imports += imports_
		}
		logics.ImportPackages = imports
		if err := genLogicFile(dir, cfg, logics, g); err != nil {
			color.Red.Printf("=======>generate error:%v\n", err)
		}
	}
	return nil
}

func genLogicByRoute(rootPkg string, route spec.Route) (string, string, string, string, string, error) {
	var (
		responseString string
		returnString   string
		requestString  string
		err            error
	)
	logic := getLogicName(route)
	imports := genLogicImports(route, rootPkg)
	if len(route.ResponseTypeName()) > 0 {
		resp := responseGoTypeName(route, typesPacket)
		responseString = "(resp " + resp + ", err error)"
		returnString = "return"
	} else {
		responseString = "error"
		returnString = "return nil"
	}
	if len(route.RequestTypeName()) > 0 {
		requestString = "req *" + requestGoTypeName(route, typesPacket)
	}
	return logic, imports, responseString, returnString, requestString, err
}

func genLogicFile(dir string, cfg *config.Config, logicInfos logicsInfo, group spec.Group) error {
	subDir := getLogicPath(group)
	goFile, err := format.FileNamingFormat(cfg.NamingFormat, logicInfos.Logic)
	if err != nil {
		return err
	}
	return genFile(fileGenConfig{
		dir:             filepath.Dir(dir),
		subdir:          subDir,
		filename:        strings.TrimSuffix(goFile, ".api") + ".go",
		templateName:    "logicTemplate",
		category:        category,
		templateFile:    logicTemplateFile,
		builtinTemplate: logicTemplate,
		data: map[string]interface{}{
			"pkgName":         logicInfos.PkgName,
			"imports":         logicInfos.ImportPackages,
			"logic":           logicInfos.Logic,
			"Logics":          logicInfos.Logics,
			"rpcPackageLower": logicInfos.rpcPackageLower,
			"rpcName":         logicInfos.rpcName,
			"useI18n":         logicInfos.useI18n,
			"useUUID":         logicInfos.useUUID,
		},
	})
}
func getLogicPath(group spec.Group) string {
	folder := group.GetAnnotation(groupProperty)
	if len(folder) == 0 {
		return logicDir
	}
	folder = strings.TrimPrefix(folder, "/")
	folder = strings.TrimSuffix(folder, "/")
	return path.Join(logicDir, folder)
}

func genLogicImports(route spec.Route, parentPkg string) string {
	var imports []string
	imports = append(imports, `"context"`+"\n")
	imports = append(imports, fmt.Sprintf("\"%s\"", pathx.JoinPackages(parentPkg, contextDir)))
	if shallImportTypesPackage(route) {
		imports = append(imports, fmt.Sprintf("\"%s\"\n", pathx.JoinPackages(parentPkg, typesDir)))
	}
	imports = append(imports, fmt.Sprintf("\"%s/core/logx\"", vars.ProjectOpenSourceURL))
	return strings.Join(imports, "\n\t")
}

func onlyPrimitiveTypes(val string) bool {
	fields := strings.FieldsFunc(val, func(r rune) bool {
		return r == '[' || r == ']' || r == ' '
	})

	for _, field := range fields {
		if field == "map" {
			continue
		}
		// ignore array dimension number, like [5]int
		if _, err := strconv.Atoi(field); err == nil {
			continue
		}
		if !api.IsBasicType(field) {
			return false
		}
	}

	return true
}

func shallImportTypesPackage(route spec.Route) bool {
	if len(route.RequestTypeName()) > 0 {
		return true
	}

	respTypeName := route.ResponseTypeName()
	if len(respTypeName) == 0 {
		return false
	}

	if onlyPrimitiveTypes(respTypeName) {
		return false
	}

	return true
}

func genExtraLogic(p *spec.ApiSpec, requestName string, method string) string {
	var logicStr string
	for _, each := range p.Types {
		if each.Name() != requestName {
			continue
		}
		defineStruct, ok := each.(spec.DefineStruct)
		if !ok {
			color.Red.Println(fmt.Sprintf("unsupported type %s", each.Name()))
			return ""
		}
		for _, member := range defineStruct.Members {
			logicStr += fmt.Sprintf("\n\t\t\t%s: req.%s,", member.Name, member.Name)
		}
	}
	if method == "GetList" {
		logicStr = strings.Replace(logicStr, "req.", "v.", -1)
	} else if method == "GetById" {
		logicStr = strings.Replace(logicStr, "req.", "data.", -1)
	}
	return logicStr
}
