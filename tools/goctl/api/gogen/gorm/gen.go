package gorm

import (
	_ "embed"
	"errors"
	"fmt"
	"github.com/gookit/color"
	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
	apiformat "github.com/zeromicro/go-zero/tools/goctl/api/format"
	apiParser "github.com/zeromicro/go-zero/tools/goctl/api/parser"
	"github.com/zeromicro/go-zero/tools/goctl/config"
	"github.com/zeromicro/go-zero/tools/goctl/pkg/golang"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	regularPerm = 0o666
	tmpFile     = "%s-%d"
)

var tmpDir = path.Join(os.TempDir(), "goctls")

type ApiLogicData struct {
	LogicName string
	LogicCode string
}

type GenGormLogicContextByAPI struct {
	ModuleName     string
	RpcName        string
	RpcPackageName string
	Dir            string
	NamingStyle    string
	Overwrite      bool
	UseCasbin      bool
	UseI18n        bool
	UseUUID        bool
	TransErr       bool
}

func (g GenGormLogicContextByAPI) Validate() error {
	if g.ModuleName == "" {
		return errors.New("please set the API service name via --api_service_name ")
	} else if g.RpcName == "" {
		return errors.New("please set the RPC service name via --rpc_name ")
	}
	return nil
}

func GenGormApiFile(g *GenGormLogicContextByAPI) error {
	allFiles, _ := getAllFile(g.Dir)
	dir := filepath.Dir(g.Dir)
	color.Green.Println("Generating...")
	if len(allFiles) < 1 {
		return errors.New("empty dir")
	}
	cfg, err := config.NewConfig(g.NamingStyle)
	if err != nil {
		return err
	}
	rootPkg, err := golang.GetParentPackage(dir)
	rootPkg = filepath.Dir(rootPkg)
	if err != nil {
		return err
	}
	var (
		middlewareStr        string
		middlewareAssignment string
		cfgImport            string
		typesVals            string
		routesVals           string
		routeImports         []collection.Set
		routeTimeOut         bool
	)
	for _, apiFile := range allFiles {
		fileName := filepath.Base(apiFile)
		if fileName == "all.api" {
			continue
		}
		api, err := apiParser.Parse(apiFile)
		if err != nil {
			color.Red.Printf("parse apifile error:%v\n ",err)
			continue
		}
		if err := api.Validate(); err != nil {
			color.Red.Printf("Validate apifile error:%v\n ",err)
			continue
		}
		// svc引入逻辑补充
		middlewareStr_, middlewareAssignment_, configImport_ := getMiddleWares(rootPkg, api)
		middlewareStr += middlewareStr_
		middlewareAssignment += middlewareAssignment_
		cfgImport += configImport_

		// types 补充
		typesVal, _ := genTypes(cfg, api)
		typesVals += typesVal

		// route 补充
		if routeStr, timeout, err := genRoutes(api); err == nil {
			routesVals += strings.TrimSpace(routeStr)
			routeImports = append(routeImports, *genRouteImports(rootPkg, api))
			routeTimeOut = timeout
		}
		logx.Must(genHandlers(dir, rootPkg, fileName, api, g))
		logx.Must(genLogics(dir, rootPkg,fileName, cfg,api, g))
		if err := backupAndSweep(apiFile); err != nil {
			return err
		}
		if err := apiformat.ApiFormatByPath(apiFile, false); err != nil {
			return err
		}
	}
	logx.Must(genServiceContext(dir, rootPkg, cfg, g, map[string]string{"configImport": cfgImport,
		"middlewareStr": middlewareStr, "middlewareAssignment": middlewareAssignment}))
	logx.Must(genTypesFile(dir, cfg, typesVals))
	logx.Must(genRouteFile(dir, cfg, routeTimeOut, routesVals, sortRouteImports(routeImports)))
	fmt.Println(color.Green.Render("Done."))
	return nil
}
