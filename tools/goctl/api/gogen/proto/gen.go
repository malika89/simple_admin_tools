package proto

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/emicklei/proto"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/iancoleman/strcase"

	"github.com/zeromicro/go-zero/tools/goctl/rpc/execx"
	"github.com/zeromicro/go-zero/tools/goctl/rpc/parser"
	"github.com/zeromicro/go-zero/tools/goctl/util/ctx"
	"github.com/zeromicro/go-zero/tools/goctl/util/entx"
	"github.com/zeromicro/go-zero/tools/goctl/util/format"
	"github.com/zeromicro/go-zero/tools/goctl/util/pathx"
	"github.com/zeromicro/go-zero/tools/goctl/util/protox"
	"github.com/zeromicro/go-zero/tools/goctl/vars"
)

const regularPerm = 0o666

// GenLogicByProtoContext describe the data used for logic generation with proto file
type GenLogicByProtoContext struct {
	ProtoDir         string
	OutputDir        string
	APIServiceName   string
	RPCServiceName   string
	RPCPbPackageName string
	Style            string
	ModelName        string
	SearchKeyNum     int
	RpcName          string
	GrpcPackage      string
	UseUUID          bool
	Multiple         bool
	HasStatus        bool
}

type ApiLogicData struct {
	LogicName string
	LogicCode string
}

func GenLogicByProto(p *GenLogicByProtoContext) error {
	outputDir, err := filepath.Abs(p.OutputDir)
	if err != nil {
		return err
	}

	logicDir := path.Join(outputDir, "internal/logic")

	protoParser := parser.NewDefaultProtoParser()
	protoData, err := protoParser.Parse(p.ProtoDir, p.Multiple)
	if err != nil {
		return err
	}

	p.RPCPbPackageName = protoData.PbPackage

	protox.ProtoField = &protox.ProtoFieldData{}

	workDir, err := filepath.Abs("./")
	if err != nil {
		return err
	}

	projectCtx, err := ctx.Prepare(workDir)
	if err != nil {
		return err
	}

	// generate logic file
	apiLogicData := GenCRUDData(p, &protoData, projectCtx)

	for _, v := range apiLogicData {
		logicFilename, err := format.FileNamingFormat(p.Style, v.LogicName)
		if err != nil {
			return err
		}

		filename := filepath.Join(logicDir, strings.ToLower(p.ModelName), logicFilename+".go")
		if err = pathx.MkdirIfNotExist(filepath.Join(logicDir, strings.ToLower(p.ModelName))); err != nil {
			return err
		}

		if pathx.FileExists(filename) {
			continue
		}

		err = os.WriteFile(filename, []byte(v.LogicCode), regularPerm)
		if err != nil {
			return err
		}
	}

	// generate api file
	apiData, err := GenApiData(p, &protoData)
	if err != nil {
		return err
	}

	apiFilePath := filepath.Join(workDir, "desc", fmt.Sprintf("%s.api", strcase.ToSnake(p.ModelName)))

	err = os.WriteFile(apiFilePath, []byte(apiData), regularPerm)
	if err != nil {
		return err
	}

	allApiFile := filepath.Join(workDir, "desc", "all.api")
	allApiData, err := os.ReadFile(allApiFile)
	if err != nil {
		return err
	}
	allApiString := string(allApiData)

	if !strings.Contains(allApiString, fmt.Sprintf("%s.api", strcase.ToSnake(p.ModelName))) {
		allApiString += fmt.Sprintf("\nimport \"%s\"", fmt.Sprintf("%s.api", strcase.ToSnake(p.ModelName)))
	}

	err = os.WriteFile(allApiFile, []byte(allApiString), regularPerm)
	if err != nil {
		return err
	}

	if runtime.GOOS == vars.OsLinux {
		_, err = execx.Run("make gen-api", workDir)
		if err != nil {
			return err
		}
	}

	return nil
}

func GenCRUDData(ctx *GenLogicByProtoContext, p *parser.Proto, projectCtx *ctx.ProjectContext) []*ApiLogicData {
	var data []*ApiLogicData
	var setLogic string

	for _, v := range p.Message {
		if strings.Contains(v.Name, ctx.ModelName) {
			if fmt.Sprintf("%sInfo", ctx.ModelName) == v.Name {
				setLogic = genSetLogic(v.Message, ctx)

				createLogic := bytes.NewBufferString("")
				createLogicTmpl, _ := template.New("createOrUpdate").Parse(createOrUpdateTpl)
				logx.Must(createLogicTmpl.Execute(createLogic, map[string]any{
					"setLogic":           setLogic,
					"modelName":          ctx.ModelName,
					"modelNameLowerCase": strings.ToLower(ctx.ModelName),
					"projectPackage":     projectCtx.Path,
					"rpcPackage":         ctx.GrpcPackage,
					"rpcName":            ctx.RpcName,
					"rpcPbPackageName":   ctx.RPCPbPackageName,
					"useUUID":            ctx.UseUUID,
				}))

				data = append(data, &ApiLogicData{
					LogicName: fmt.Sprintf("CreateOrUpdate%sLogic", ctx.ModelName),
					LogicCode: createLogic.String(),
				})

				// delete logic
				deleteLogic := bytes.NewBufferString("")
				deleteLogicTmpl, _ := template.New("delete").Parse(deleteLogicTpl)
				logx.Must(deleteLogicTmpl.Execute(deleteLogic, map[string]any{
					"setLogic":           setLogic,
					"modelName":          ctx.ModelName,
					"modelNameLowerCase": strings.ToLower(ctx.ModelName),
					"projectPackage":     projectCtx.Path,
					"rpcPackage":         ctx.GrpcPackage,
					"rpcName":            ctx.RpcName,
					"rpcPbPackageName":   ctx.RPCPbPackageName,
					"useUUID":            ctx.UseUUID,
				}))

				data = append(data, &ApiLogicData{
					LogicName: fmt.Sprintf("Delete%sLogic", ctx.ModelName),
					LogicCode: deleteLogic.String(),
				})

				// batch delete logic
				batchDeleteLogic := bytes.NewBufferString("")
				batchDeleteLogicTmpl, _ := template.New("batchDelete").Parse(batchDeleteLogicTpl)
				logx.Must(batchDeleteLogicTmpl.Execute(batchDeleteLogic, map[string]any{
					"setLogic":           setLogic,
					"modelName":          ctx.ModelName,
					"modelNameLowerCase": strings.ToLower(ctx.ModelName),
					"projectPackage":     projectCtx.Path,
					"rpcPackage":         ctx.GrpcPackage,
					"rpcName":            ctx.RpcName,
					"rpcPbPackageName":   ctx.RPCPbPackageName,
					"useUUID":            ctx.UseUUID,
				}))

				data = append(data, &ApiLogicData{
					LogicName: fmt.Sprintf("BatchDelete%sLogic", ctx.ModelName),
					LogicCode: batchDeleteLogic.String(),
				})

				if ctx.HasStatus {
					// update status logic
					updateStatusLogic := bytes.NewBufferString("")
					updateStatusLogicTmpl, _ := template.New("update_status").Parse(updateStatusLogicTpl)
					logx.Must(updateStatusLogicTmpl.Execute(updateStatusLogic, map[string]any{
						"setLogic":           setLogic,
						"modelName":          ctx.ModelName,
						"modelNameLowerCase": strings.ToLower(ctx.ModelName),
						"projectPackage":     projectCtx.Path,
						"rpcPackage":         ctx.GrpcPackage,
						"rpcName":            ctx.RpcName,
						"rpcPbPackageName":   ctx.RPCPbPackageName,
						"useUUID":            ctx.UseUUID,
					}))

					data = append(data, &ApiLogicData{
						LogicName: fmt.Sprintf("Update%sStatusLogic", ctx.ModelName),
						LogicCode: updateStatusLogic.String(),
					})
				}
			}

			if fmt.Sprintf("%sListReq", ctx.ModelName) == v.Name {
				searchLogic := strings.Builder{}
				for _, field := range v.Elements {
					field.Accept(protox.MessageVisitor{})
					if protox.ProtoField.Name == "page" || protox.ProtoField.Name == "page_size" {
						continue
					}
					searchLogic.WriteString(fmt.Sprintf("\n\t\t\t%s: req.%s,", parser.CamelCase(protox.ProtoField.Name),
						parser.CamelCase(protox.ProtoField.Name)))
				}

				if setLogic == "" {
					for _, m := range p.Message {
						if strings.Contains(m.Name, ctx.ModelName) {
							if fmt.Sprintf("%sInfo", ctx.ModelName) == m.Name {
								setLogic = genSetLogic(m.Message, ctx)
							}
						}
					}
				}

				getListLogic := bytes.NewBufferString("")
				getListLogicTmpl, _ := template.New("getList").Parse(getListLogicTpl)
				logx.Must(getListLogicTmpl.Execute(getListLogic, map[string]any{
					"setLogic":           strings.Replace(setLogic, "req.", "v.", -1),
					"modelName":          ctx.ModelName,
					"modelNameLowerCase": strings.ToLower(ctx.ModelName),
					"projectPackage":     projectCtx.Path,
					"rpcPackage":         ctx.GrpcPackage,
					"rpcName":            ctx.RpcName,
					"rpcPbPackageName":   ctx.RPCPbPackageName,
					"searchKeys":         searchLogic.String(),
					"useUUID":            ctx.UseUUID,
				}))

				data = append(data, &ApiLogicData{
					LogicName: fmt.Sprintf("Get%sListLogic", ctx.ModelName),
					LogicCode: getListLogic.String(),
				})
			}

		}
	}

	return data
}

func GenApiData(ctx *GenLogicByProtoContext, p *parser.Proto) (string, error) {
	infoData := strings.Builder{}
	listData := strings.Builder{}
	hasStatus := false
	var data string

	for _, v := range p.Message {
		if strings.Contains(v.Name, ctx.ModelName) {
			if fmt.Sprintf("%sInfo", ctx.ModelName) == v.Name {
				for _, field := range v.Elements {
					field.Accept(protox.MessageVisitor{})
					if entx.IsBaseProperty(protox.ProtoField.Name) {
						continue
					} else if protox.ProtoField.Name == "status" {
						hasStatus = true
					}

					var structData string

					structData = fmt.Sprintf("\n\n        // %s\n        %s  %s `json:\"%s\"`",
						parser.CamelCase(protox.ProtoField.Name),
						parser.CamelCase(protox.ProtoField.Name),
						entx.ConvertProtoTypeToGoType(protox.ProtoField.Type),
						strcase.ToLowerCamel(protox.ProtoField.Name))

					infoData.WriteString(structData)
				}
			} else if strings.HasSuffix(v.Name, "ListReq") {
				for _, field := range v.Elements {
					field.Accept(protox.MessageVisitor{})

					var structData string

					structData = fmt.Sprintf("\n\n        // %s\n        %s  %s `json:\"%s,optional\"`",
						parser.CamelCase(protox.ProtoField.Name),
						parser.CamelCase(protox.ProtoField.Name),
						entx.ConvertProtoTypeToGoType(protox.ProtoField.Type),
						strcase.ToLowerCamel(protox.ProtoField.Name))

					if protox.ProtoField.Type == "string" {
						listData.WriteString(structData)
					}
				}
			}
		}
	}

	apiTemplateData := bytes.NewBufferString("")
	apiTmpl, _ := template.New("apiTpl").Parse(apiTpl)
	logx.Must(apiTmpl.Execute(apiTemplateData, map[string]any{
		"infoData":           infoData.String(),
		"modelName":          ctx.ModelName,
		"modelNameLowerCase": strings.Replace(strcase.ToSnake(ctx.ModelName), "_", " ", -1),
		"modelNameSnake":     strcase.ToSnake(ctx.ModelName),
		"listData":           listData.String(),
		"apiServiceName":     ctx.APIServiceName,
		"useUUID":            ctx.UseUUID,
		"hasStatus":          hasStatus,
	}))
	data = apiTemplateData.String()

	return data, nil
}

func genSetLogic(v *proto.Message, ctx *GenLogicByProtoContext) string {
	var setLogic strings.Builder
	for _, field := range v.Elements {
		field.Accept(protox.MessageVisitor{})
		if entx.IsBaseProperty(protox.ProtoField.Name) {
			if protox.ProtoField.Name == "id" && protox.ProtoField.Type == "string" {
				ctx.UseUUID = true
			}
			continue
		}

		if protox.ProtoField.Name == "status" {
			ctx.HasStatus = true
		}

		setLogic.WriteString(fmt.Sprintf("\n        \t%s: req.%s,", parser.CamelCase(protox.ProtoField.Name),
			parser.CamelCase(protox.ProtoField.Name)))
	}
	return setLogic.String()
}
