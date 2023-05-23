// Copyright 2023 The Ryan SU Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gorm

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"github.com/zeromicro/go-zero/tools/goctl/util"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
	"github.com/smallnest/gen/dbmeta"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/tools/goctl/rpc/parser"
	"github.com/zeromicro/go-zero/tools/goctl/util/console"
	"github.com/zeromicro/go-zero/tools/goctl/util/ctx"
	"github.com/zeromicro/go-zero/tools/goctl/util/entx"
	"github.com/zeromicro/go-zero/tools/goctl/util/format"
	"github.com/zeromicro/go-zero/tools/goctl/util/pathx"
	"github.com/zeromicro/go-zero/tools/goctl/util/protox"
)

const regularPerm = 0o666

type RpcLogicData struct {
	LogicName string
	LogicCode string
}

type GenGormLogicContext struct {
	Schema          string
	Output          string
	Style           string
	ProtoFieldStyle string
	SearchKeyNum    int
	ProjectName     string
	UseUUID         bool
	UseI18n         bool
	ProtoOut        string
	Overwrite       bool
}

func (g GenGormLogicContext) Validate() error {
	if g.Schema == "" {
		return errors.New("the schema dir cannot be empty ")
	} else if !strings.HasSuffix(g.Schema, "config.json") {
		return errors.New("please input correct schema directory e.g. ./gorm/schema/config.json ")
	}
	return nil
}

// GenGormLogic generates the gorm CRUD logic files of the rpc service.
func GenGormLogic(g *GenGormLogicContext) error {
	return genGormLogic(g)
}

func genGormLogic(g *GenGormLogicContext) error {
	outputDir, err := filepath.Abs(g.Output)
	if err != nil {
		return err
	}

	var logicDir string

	logicDir = path.Join(outputDir, "internal/logic")

	workDir, err := filepath.Abs("./")
	if err != nil {
		return err
	}

	projectCtx, err := ctx.Prepare(workDir)
	if err != nil {
		return err
	}

	schemas, cfg, err := util.LoadTableSchemas(g.Schema)
	if err != nil {
		return err
	}
	if err = cfg.LocalLoadTemplate(schemas, outputDir); err != nil {
		fmt.Printf("write struct error:%v with outputDir:%s", err,outputDir)
	}
	for _, s := range schemas {
		rpcLogicData := GenCRUDData(g, projectCtx, s)
		logicFilename, err := format.FileNamingFormat(g.Style, rpcLogicData.LogicName)
		if err != nil {
			return err
		}
		// group
		if err = pathx.MkdirIfNotExist(filepath.Join(logicDir, s.PackageName)); err != nil {
			return err
		}
		filename := filepath.Join(logicDir, s.PackageName, logicFilename+".go")
		if pathx.FileExists(filename) && !g.Overwrite {
			continue
		}
		err = os.WriteFile(filename, []byte(rpcLogicData.LogicCode), regularPerm)
		if err != nil {
			return err
		}

		// generate proto file
		protoMessage, protoFunctions, err := GenProtoData(s, g)
		if err != nil {
			return err
		}
		if err = pathx.MkdirIfNotExist(filepath.Join(outputDir, "desc")); err != nil {
			return err
		}
		protoFileName := filepath.Join(outputDir, "desc", s.StructName+".proto")
		if !pathx.FileExists(protoFileName) || g.Overwrite {
			err = os.WriteFile(protoFileName, []byte(fmt.Sprintf("syntax = \"proto3\";\n\nservice %s {\n}",
				strcase.ToCamel(g.ProjectName))), os.ModePerm)
			if err != nil {
				return fmt.Errorf("failed to create proto file : %s", err.Error())
			}
		}
		protoFileData, err := os.ReadFile(protoFileName)
		if err != nil {
			return err
		}

		protoDataString := string(protoFileData)

		if strings.Contains(protoDataString, protoMessage) || strings.Contains(protoDataString, protoFunctions) {
			continue
		}

		// generate new proto file
		newProtoData := strings.Builder{}
		serviceBeginIndex, _, serviceEndIndex := protox.FindBeginEndOfService(protoDataString, strcase.ToCamel(s.PackageName))
		if serviceBeginIndex == -1 {
			continue
		}
		newProtoData.WriteString(protoDataString[:serviceBeginIndex-1])
		newProtoData.WriteString(fmt.Sprintf("\n// %s message\n\n", s.StructName))
		newProtoData.WriteString(fmt.Sprintf("%s\n", protoMessage))
		newProtoData.WriteString(protoDataString[serviceBeginIndex-1 : serviceEndIndex-1])
		newProtoData.WriteString(fmt.Sprintf("\n\n  // %s management\n", s.StructName))
		newProtoData.WriteString(fmt.Sprintf("%s\n", protoFunctions))
		newProtoData.WriteString(protoDataString[serviceEndIndex-1:])

		err = os.WriteFile(protoFileName, []byte(newProtoData.String()), regularPerm)
		if err != nil {
			return err
		}
	}

	console.NewColorConsole(true).Success("Generate Gorm Logic files for RPC successfully")
	return nil
}

func GenCRUDData(g *GenGormLogicContext, projectCtx *ctx.ProjectContext, schema *dbmeta.ModelInfo) *RpcLogicData {
	var data = &RpcLogicData{}
	hasTime, hasUUID := false, false
	// end string means whether to use \n
	endString := ""
	packageName := schema.PackageName

	setLogic := strings.Builder{}
	for _, v := range schema.DBMeta.Columns() {
		colName := v.Name()
		colType := v.ColumnType()
		if colName == "id" {
			if entx.IsUUIDType(colType) {
				g.UseUUID = true
			}
			continue
		} else if entx.IsOnlyEntType(colType) {
			setLogic.WriteString(fmt.Sprintf("\t\t\t%s:\tin.%s,\n", parser.CamelCase(colName),
				parser.CamelCase(colName)))
		} else {
			if entx.IsTimeProperty(colType) {
				hasTime = true
				setLogic.WriteString(fmt.Sprintf("\t\t\t%s:\ttime.Unix(in.%s, 0),\n", parser.CamelCase(colName),
					parser.CamelCase(colName)))
			} else if entx.IsUpperProperty(colName) {
				if entx.IsGoTypeNotPrototype(colType) {
					if colType == "[16]byte" {
						setLogic.WriteString(fmt.Sprintf("\t\t\t%s:\tuuidx.ParseUUIDString(in.%s),\n", entx.ConvertSpecificNounToUpper(colName),
							parser.CamelCase(colName)))
						hasUUID = true
					} else {
						setLogic.WriteString(fmt.Sprintf("\t\t\t%s:\t%s(in.%s),\n", entx.ConvertSpecificNounToUpper(colName),
							colType, parser.CamelCase(colName)))
					}
				} else {
					setLogic.WriteString(fmt.Sprintf("\t\t\t%s:\tin.%s,\n", entx.ConvertSpecificNounToUpper(colName),
						parser.CamelCase(colName)))
				}
			} else {
				if entx.IsGoTypeNotPrototype(colType) {
					setLogic.WriteString(fmt.Sprintf("\t\t\t%s:\t%s(in.%s),\n", parser.CamelCase(colName),
						colType, parser.CamelCase(colName)))
				} else {
					setLogic.WriteString(fmt.Sprintf("\t\t\t%s:\tin.%s,\n", parser.CamelCase(colName),
						parser.CamelCase(colName)))
				}
			}
		}
	}
	createLogic := bytes.NewBufferString("")
	logicTmpl, _ := template.New("create").Parse(createTpl)
	_ = logicTmpl.Execute(createLogic, map[string]any{
		"hasTime":     hasTime,
		"hasUUID":     hasUUID,
		"setLogic":    setLogic.String(),
		"modelName":   schema.StructName,
		"projectName": strings.ToLower(g.ProjectName),
		"projectPath": projectCtx.Path,
		"packageName": packageName,
		"useUUID":     g.UseUUID, // UUID primary key
		"useI18n":     g.UseI18n,
	})
	data.LogicName = fmt.Sprintf("%sLogic", schema.StructName)
	data.LogicCode = createLogic.String()
	updateLogic := bytes.NewBufferString("")
	updateLogicTmpl, _ := template.New("update").Parse(updateTpl)
	_ = updateLogicTmpl.Execute(updateLogic, map[string]any{
		"hasTime":     hasTime,
		"hasUUID":     hasUUID,
		"setLogic":    setLogic.String(),
		"modelName":   schema.StructName,
		"projectName": strings.ToLower(g.ProjectName),
		"projectPath": projectCtx.Path,
		"packageName": packageName,
		"useUUID":     g.UseUUID, // UUID primary key
		"useI18n":     g.UseI18n,
	})

	data.LogicCode += fmt.Sprintf("\n%s", updateLogic.String())
	predicateData := strings.Builder{}
	predicateData.WriteString(fmt.Sprintf("\tvar predicates []predicate.%s\n", schema.StructName))
	count := 0
	for _, v := range schema.DBMeta.Columns() {
		if v.ColumnType() == "string" && !strings.Contains(strings.ToLower(v.Name()), "uuid") &&
			count < g.SearchKeyNum && v.Name() != "id" {
			camelName := parser.CamelCase(v.Name())
			predicateData.WriteString(fmt.Sprintf("\tif in.%s != \"\" {\n\t\tpredicates = append(predicates, %s.%sContains(in.%s))\n\t}\n",
				camelName, strings.ToLower(schema.StructName), entx.ConvertSpecificNounToUpper(v.Name()), camelName))
			count++
		}
	}
	predicateData.WriteString(fmt.Sprintf("\tresult, err := l.svcCtx.DB.%s.Query().Where(predicates...).Page(l.ctx, in.Page, in.PageSize)",
		schema.StructName))

	listData := strings.Builder{}

	for i, v := range schema.DBMeta.Columns() {
		if v.Name() == "id" {
			continue
		} else {
			nameCamelCase := parser.CamelCase(v.Name())

			if i < (len(schema.Fields) - 1) {
				endString = "\n"
			} else {
				endString = ""
			}

			if entx.IsUUIDType(v.ColumnType()) {
				listData.WriteString(fmt.Sprintf("\t\t\t%s:\tv.%s.String(),%s", nameCamelCase,
					entx.ConvertSpecificNounToUpper(nameCamelCase), endString))
			} else if entx.IsOnlyEntType(v.ColumnType()) {
				listData.WriteString(fmt.Sprintf("\t\t\t%s:\t%s(v.%s),%s", nameCamelCase,
					entx.ConvertOnlyEntTypeToGoType(v.ColumnType()),
					entx.ConvertSpecificNounToUpper(nameCamelCase), endString))
			} else if entx.IsTimeProperty(v.ColumnType()) {
				listData.WriteString(fmt.Sprintf("\t\t\t%s:\tv.%s.UnixMilli(),%s", nameCamelCase,
					entx.ConvertSpecificNounToUpper(nameCamelCase), endString))
			} else {
				if entx.IsUpperProperty(v.Name()) {
					if entx.IsGoTypeNotPrototype(v.ColumnType()) {
						listData.WriteString(fmt.Sprintf("\t\t\t%s:\t%s(v.%s),%s", nameCamelCase,
							entx.ConvertEntTypeToGotype(v.ColumnType()), entx.ConvertSpecificNounToUpper(v.Name()), endString))
					} else {
						listData.WriteString(fmt.Sprintf("\t\t\t%s:\tv.%s,%s", nameCamelCase,
							entx.ConvertSpecificNounToUpper(v.Name()), endString))
					}
				} else {
					if entx.IsGoTypeNotPrototype(v.ColumnType()) {
						listData.WriteString(fmt.Sprintf("\t\t\t%s:\t%s(v.%s),%s", nameCamelCase,
							entx.ConvertEntTypeToGotype(v.ColumnType()), nameCamelCase, endString))
					} else {
						listData.WriteString(fmt.Sprintf("\t\t\t%s:\tv.%s,%s", nameCamelCase,
							nameCamelCase, endString))
					}
				}
			}
		}
	}

	getListLogic := bytes.NewBufferString("")
	getListLogicTmpl, _ := template.New("getList").Parse(getListLogicTpl)
	_ = getListLogicTmpl.Execute(getListLogic, map[string]any{
		"modelName":   schema.StructName,
		"listData":    listData.String(),
		"projectName": strings.ToLower(g.ProjectName),
		"packageName": packageName,
		"useUUID":     g.UseUUID,
	})
	data.LogicCode += fmt.Sprintf("\n%s", getListLogic.String())

	getByIdLogic := bytes.NewBufferString("")
	getByIdLogicTmpl, _ := template.New("getById").Parse(getByIdLogicTpl)
	_ = getByIdLogicTmpl.Execute(getByIdLogic, map[string]any{
		"modelName":   schema.StructName,
		"listData":    strings.Replace(listData.String(), "v.", "result.", -1),
		"projectName": strings.ToLower(g.ProjectName),
		"packageName": packageName,
		"useUUID":     g.UseUUID,
	})

	data.LogicCode += fmt.Sprintf("\n%s", getByIdLogic.String())

	deleteLogic := bytes.NewBufferString("")
	deleteLogicTmpl, _ := template.New("delete").Parse(deleteLogicTpl)
	err := deleteLogicTmpl.Execute(deleteLogic, map[string]any{
		"modelName":   schema.StructName,
		"projectName": strings.ToLower(g.ProjectName),
		"packageName": packageName,
		"useUUID":     g.UseUUID,
		"useI18n":     g.UseI18n,
	})
	if err != nil {
		fmt.Println(err)
	}
	data.LogicCode += fmt.Sprintf("\n%s", deleteLogic.String())

	return data
}

func GenProtoData(schema *dbmeta.ModelInfo, g *GenGormLogicContext) (string, string, error) {
	var protoMessage strings.Builder
	schemaNameCamelCase := parser.CamelCase(schema.StructName)
	// hasStatus means it has status field
	hasStatus := false
	// end string means whether to use \n
	endString := ""
	// info message
	idString, _ := format.FileNamingFormat(g.ProtoFieldStyle, "id")
	protoMessage.WriteString(fmt.Sprintf("message %sInfo {\n  %s %s = 1;\n", schemaNameCamelCase, entx.ConvertIDType(g.UseUUID), idString))
	index := 2
	for i, v := range schema.DBMeta.Columns() {
		if v.Name() == "id" {
			continue
		} else if v.Name() == "status" {
			statusString, _ := format.FileNamingFormat(g.ProtoFieldStyle, v.Name())
			protoMessage.WriteString(fmt.Sprintf("  uint32 %s = %d;\n", statusString, index))
			hasStatus = true
			index++
		} else {
			if i < (len(schema.Fields) - 1) {
				endString = "\n"
			} else {
				endString = ""
			}

			formatedString, _ := format.FileNamingFormat(g.ProtoFieldStyle, v.Name())
			if entx.IsTimeProperty(v.ColumnType()) {
				protoMessage.WriteString(fmt.Sprintf("  int64  %s = %d;%s", formatedString, index, endString))
			} else {
				protoMessage.WriteString(fmt.Sprintf("  %s %s = %d;%s", entx.ConvertEntTypeToProtoType(v.ColumnType()),
					formatedString, index, endString))
			}

			if i == (len(schema.Fields) - 1) {
				protoMessage.WriteString("\n}\n\n")
			}

			index++
		}
	}

	// List message
	totalString, _ := format.FileNamingFormat(g.ProtoFieldStyle, "total")
	dataString, _ := format.FileNamingFormat(g.ProtoFieldStyle, "data")
	protoMessage.WriteString(fmt.Sprintf("message %sListResp {\n  uint64 %s = 1;\n  repeated %sInfo %s = 2;\n}\n\n",
		schemaNameCamelCase, totalString, schemaNameCamelCase, dataString))

	// List Request message
	pageString, _ := format.FileNamingFormat(g.ProtoFieldStyle, "page")
	pageSizeString, _ := format.FileNamingFormat(g.ProtoFieldStyle, "page_size")
	filterString, _ := format.FileNamingFormat(g.ProtoFieldStyle, "filter")
	protoMessage.WriteString(fmt.Sprintf("message %sListReq {\n  uint64 %s = 1;\n  uint64 %s = 2;\n  map<string, string> %s =3;\n",
		schemaNameCamelCase, pageString, pageSizeString, filterString))
	count := 0
	index = 3

	for i, v := range schema.DBMeta.Columns() {
		if v.ColumnType() == "string" && !strings.Contains(strings.ToLower(v.Name()), "uuid") && count < g.SearchKeyNum {
			if i < len(schema.Fields) && count < g.SearchKeyNum {
				formatedString, _ := format.FileNamingFormat(g.ProtoFieldStyle, v.Name())
				protoMessage.WriteString(fmt.Sprintf("  %s %s = %d;\n", entx.ConvertEntTypeToProtoType(v.ColumnType()),
					formatedString, index))
				index++
				count++
			}
		}

		if i == (len(schema.Fields) - 1) {
			protoMessage.WriteString("}\n")
		}
	}

	// group
	groupName := fmt.Sprintf("  // group: %s\n", schema.PackageName)

	protoRpcFunction := bytes.NewBufferString("")
	protoTmpl, err := template.New("proto").Parse(protoTpl)
	err = protoTmpl.Execute(protoRpcFunction, map[string]any{
		"modelName": schema.StructName,
		"groupName": groupName,
		"useUUID":   g.UseUUID,
		"hasStatus": hasStatus,
	})

	if err != nil {
		logx.Error(err)
		return "", "", err
	}

	return protoMessage.String(), protoRpcFunction.String(), nil
}
