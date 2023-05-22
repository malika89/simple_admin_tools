package util

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/bxcodec/faker/v3"
	"github.com/gobuffalo/packr/v2"
	"github.com/iancoleman/strcase"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	dynamicstruct "github.com/ompluscator/dynamic-struct"
	"github.com/smallnest/gen/dbmeta"
	"github.com/zeromicro/go-zero/tools/goctl/util/pathx"
	"io/ioutil"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var parsePrimaryKeys = map[string]string{
	"uint8":     "parseUint8",
	"uint16":    "parseUint16",
	"uint32":    "parseUint32",
	"uint64":    "parseUint64",
	"int":       "parseInt",
	"int8":      "parseInt8",
	"int16":     "parseInt16",
	"int32":     "parseInt32",
	"int64":     "parseInt64",
	"string":    "parseString",
	"uuid.UUID": "parseUUID",
	"time.Time": "parseTime",
	"varbinary": "parseBytes",
}

type GormConfig struct {
	SQLType        string              `json:"sql_type"`
	SQLConnStr     string              `json:"sql_conn_str"`
	SQLDatabase    string              `json:"sql_database"`
	AddJSONTag     bool                `json:"add_json_tag"`
	AddGormTag     bool                `json:"add_gorm_tag"`
	JSONNameFormat string              `json:"json_name_format"`
	ProjectName    string              `json:"project_name"`
	Tables         []map[string]string `json:"tables"`
	DB             *sql.DB
}

func LoadTableSchemas(path string) (tables map[string]*dbmeta.ModelInfo, err error) {
	var cfg GormConfig
	if err := loadMapping(); err != nil {
		return nil, err
	}
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}
	if err = json.Unmarshal(content, &cfg); err != nil {
		return nil, err
	}
	db, err := sql.Open(cfg.SQLType, cfg.SQLConnStr)
	if err != nil {
		fmt.Printf("Error in open database: %v\n\n", err.Error())
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		fmt.Printf("Error pinging database: %v\n\n", err.Error())
		return nil, err
	}
	cfg.DB = db

	tableInfos := cfg.loadTableInfo()
	if err = cfg.loadTemplate(tableInfos); err != nil {
		fmt.Printf("write struct error:%v", err)
	}
	return tableInfos, nil
}
func (conf *GormConfig) loadTableInfo() map[string]*dbmeta.ModelInfo {

	tableInfos := make(map[string]*dbmeta.ModelInfo)
	var (
		tableIdx = 0
		dbTables []string
	)
	for _, v := range conf.Tables {
		dbTables = append(dbTables, v["table_name"])
	}

	for _, tableInfo := range conf.Tables {
		tableName := tableInfo["table_name"]
		if strings.HasPrefix(tableName, "[") && strings.HasSuffix(tableName, "]") {
			tableName = tableName[1 : len(tableName)-1]
		}
		dbMeta, err := dbmeta.LoadMeta(conf.SQLType, conf.DB, conf.SQLDatabase, tableName)
		if err != nil {
			fmt.Printf("Warning - LoadMeta skipping table info for %s error: %v\n", tableName, err)
			continue
		}
		packageName := tableInfo["package_name"]
		structName := tableInfo["struct_name"]
		modelInfo, err := conf.generateModelInfo(dbMeta, tableName, packageName, structName)
		if err != nil {
			fmt.Printf("Error - %v\n", err)
			continue
		}

		modelInfo.Index = tableIdx
		modelInfo.IndexPlus1 = tableIdx + 1
		tableIdx++

		tableInfos[tableName] = modelInfo
	}

	return tableInfos
}

func (conf *GormConfig) generateModelInfo(dbMeta dbmeta.DbTableMeta, tableName, packageName, structName string) (*dbmeta.ModelInfo, error) {

	fields, err := generateFieldsTypes(dbMeta.Columns(), tableName, conf.JSONNameFormat, conf.AddJSONTag, conf.AddGormTag)
	if err != nil {
		return nil, err
	}

	generator := dynamicstruct.NewStruct()

	noOfPrimaryKeys := 0
	for i, c := range fields {
		meta := dbMeta.Columns()[i]
		jsonName := formatFieldName(conf.JSONNameFormat, meta.Name())
		tag := fmt.Sprintf(`json:"%s"`, jsonName)
		fakeData := c.FakeData
		generator = generator.AddField(c.GoFieldName, fakeData, tag)
		if meta.IsPrimaryKey() {
			c.PrimaryKeyArgName = fmt.Sprintf("arg%s", dbmeta.FmtFieldName(c.GoFieldName))
			noOfPrimaryKeys++
		}
	}

	instance := generator.Build().New()

	err = faker.FakeData(&instance)
	if err != nil {
		fmt.Println(err)
	}

	var code []string
	for _, f := range fields {

		if f.PrimaryKeyFieldParser == "unsupported" {
			return nil, fmt.Errorf("unable to generate code for table: %s, primary key column: [%d] %s has unsupported type: %s / %s",
				dbMeta.TableName(), f.ColumnMeta.Index(), f.ColumnMeta.Name(), f.ColumnMeta.DatabaseTypeName(), f.GoFieldType)
		}
		code = append(code, f.Code)
	}

	var modelInfo = &dbmeta.ModelInfo{
		PackageName:     packageName,
		StructName:      structName,
		TableName:       tableName,
		ShortStructName: strings.ToLower(string(structName[0])),
		Fields:          code,
		CodeFields:      fields,
		DBMeta:          dbMeta,
		Instance:        instance,
	}

	return modelInfo, nil
}

func (conf *GormConfig) loadTemplate(tables map[string]*dbmeta.ModelInfo) error {
	var (
		ModelTmpl *dbmeta.GenTemplate
		err       error
	)
	genConf := dbmeta.Config{
		SQLType:             conf.SQLType,
		SQLConnStr:          conf.SQLConnStr,
		SQLDatabase:         conf.SQLDatabase,
		Module:              conf.ProjectName,
		AddJSONAnnotation:   conf.AddJSONTag,
		AddGormAnnotation:   conf.AddGormTag,
		JSONNameFormat:      conf.JSONNameFormat,
		OutDir:              filepath.Join(),
		Overwrite:           true,
		FileNamingTemplate:  "{{.}}",
		ModelNamingTemplate: "{{FmtFieldName .}}",
		FieldNamingTemplate: "{{FmtFieldName (stringifyFirstChar .) }}",
		ContextMap:          map[string]interface{}{"tables": tables},
		TemplateLoader:      LoadTemplate,
		TableInfos:          tables,
	}
	_, currentPath, _, ok := runtime.Caller(1)
	if !ok {
		return fmt.Errorf("Can not get current file info")
	}
	currentDir := filepath.Dir(currentPath)
	fileName := filepath.Join(filepath.Dir(currentDir), "rpc", "generator", "gorm", "model.go.tpl")
	if ModelTmpl, err = genConf.TemplateLoader(fileName); err != nil {
		fmt.Printf("Error loading template %v\n", err)
		return err
	}
	workDir, _ := filepath.Abs("/")
	for _, table := range tables {
		modelDir := filepath.Join(workDir, "gorm", table.PackageName)
		if err = pathx.MkdirIfNotExist(modelDir); err != nil {
			return err
		}
		modelInfo := genConf.CreateContextForTableFile(table)
		modelFile := filepath.Join(modelDir, fmt.Sprintf("%s.go", table.TableName))
		err = genConf.WriteTemplate(ModelTmpl, modelInfo, modelFile)
		if err != nil {
			fmt.Printf("Error writing file: %v\n", err)
			continue
		}
	}

	return nil
}

func generateFieldsTypes(columns []dbmeta.ColumnMeta, tableName, jsonFormat string, addJsonTag, addGormTag bool) ([]*dbmeta.FieldInfo, error) {

	var fields []*dbmeta.FieldInfo
	field := ""
	for i, col := range columns {
		fieldName := col.Name()

		fi := &dbmeta.FieldInfo{
			Index: i,
		}
		columnTypeRevised := strings.ToLower(col.DatabaseTypeName())
		if columnTypeRevised == "uunsigned bigint" {
			columnTypeRevised = "ubigint"
		} else if columnTypeRevised == "uunsigned tinyint" {
			columnTypeRevised = "utinyint"

		}
		valueType, err := dbmeta.SQLTypeToGoType(strings.ReplaceAll(columnTypeRevised, "uunsigned", "unsigned"), col.Nullable(), false)
		if err != nil { // unknown type
			fmt.Printf("table: %s unable to generate struct field: %s type: %s error: %v\n", tableName, fieldName, col.DatabaseTypeName(), err)
			continue
		}

		fieldName = dbmeta.Replace("{{FmtFieldName (stringifyFirstChar .) }}", fieldName)

		fi.GormAnnotation = createGormAnnotation(col)
		fi.JSONAnnotation = createJSONAnnotation(jsonFormat, fieldName)

		var annotations []string
		if addGormTag {
			annotations = append(annotations, fi.GormAnnotation)
		}

		if addJsonTag {
			annotations = append(annotations, fi.JSONAnnotation)
		}
		gogoTags := []string{fi.GormAnnotation, fi.JSONAnnotation, fi.DBAnnotation}
		GoGoMoreTags := strings.Join(gogoTags, " ")
		if len(annotations) > 0 {
			field = fmt.Sprintf("%s %s `%s`",
				fieldName,
				valueType,
				strings.Join(annotations, " "))
		} else {
			field = fmt.Sprintf("%s %s", fieldName, valueType)
		}

		field = fmt.Sprintf("//%s\n    %s", col.String(), field)
		if col.Comment() != "" {
			field = fmt.Sprintf("%s // %s", field, col.Comment())
		}

		sqlMapping, _ := dbmeta.SQLTypeToMapping(columnTypeRevised)
		goType, _ := dbmeta.SQLTypeToGoType(columnTypeRevised, false, false)
		protobufType, _ := dbmeta.SQLTypeToProtobufType(col.DatabaseTypeName())
		fakeData := createFakeData(goType, fieldName)
		primaryKeyFieldParser := ""
		if col.IsPrimaryKey() {
			var ok bool
			primaryKeyFieldParser, ok = parsePrimaryKeys[goType]
			if !ok {
				primaryKeyFieldParser = "unsupported"
			}
		}

		fi.Code = field
		fi.GoFieldName = fieldName
		fi.GoFieldType = valueType
		fi.Comment = col.String()
		fi.GoAnnotations = annotations
		fi.JSONFieldName = formatFieldName(jsonFormat, col.Name())
		fi.ProtobufType = protobufType
		fi.ProtobufPos = i + 1
		fi.ColumnMeta = col
		fi.FakeData = fakeData
		fi.PrimaryKeyFieldParser = primaryKeyFieldParser
		fi.SQLMapping = sqlMapping
		fi.GoGoMoreTags = GoGoMoreTags
		fields = append(fields, fi)
	}
	return fields, nil
}

func formatFieldName(nameFormat string, name string) string {

	var jsonName string
	switch nameFormat {
	case "snake":
		jsonName = strcase.ToSnake(name)
	case "camel":
		jsonName = strcase.ToCamel(name)
	case "lower_camel":
		jsonName = strcase.ToLowerCamel(name)
	case "none":
		jsonName = name
	default:
		jsonName = name
	}
	return jsonName
}

func createJSONAnnotation(nameFormat, columnName string) string {
	name := formatFieldName(nameFormat, columnName)
	return fmt.Sprintf("json:\"%s\"", name)
}

func createGormAnnotation(c dbmeta.ColumnMeta) string {
	buf := bytes.Buffer{}

	key := c.Name()
	buf.WriteString("gorm:\"")

	if c.IsPrimaryKey() {
		buf.WriteString("primary_key;")
	}
	if c.IsAutoIncrement() {
		buf.WriteString("AUTO_INCREMENT;")
	}

	buf.WriteString("column:")
	buf.WriteString(key)
	buf.WriteString(";")

	if c.DatabaseTypeName() != "" {
		buf.WriteString("type:")
		buf.WriteString(c.DatabaseTypeName())
		buf.WriteString(";")

		if c.ColumnLength() > 0 {
			buf.WriteString(fmt.Sprintf("size:%d;", c.ColumnLength()))
		}

		if c.DefaultValue() != "" {
			value := c.DefaultValue()
			value = strings.Replace(value, "\"", "'", -1)

			if value == "NULL" || value == "null" {
				value = ""
			}

			if value != "" && !strings.Contains(value, "()") {
				buf.WriteString(fmt.Sprintf("default:%s;", value))
			}
		}

	}

	buf.WriteString("\"")
	return buf.String()
}

func loadMapping() error {
	_, currentPath, _, ok := runtime.Caller(1)
	if !ok {
		return fmt.Errorf("Can not get current file info")
	}
	currentDir := filepath.Dir(currentPath)
	content, err := ioutil.ReadFile(filepath.Join(currentDir, "mapping.json"))
	if err != nil {
		return err
	}
	err = dbmeta.ProcessMappings("internal", content, false)
	if err != nil {
		fmt.Printf("Error processing default mapping file error: %v\n", err)
		return err
	}
	return nil
}

func createFakeData(valueType string, name string) interface{} {

	switch valueType {
	case "[]byte":
		return []byte("hello world")
	case "bool":
		return true
	case "float32":
		return float32(1.0)
	case "float64":
		return float64(1.0)
	case "int":
		return int(1)
	case "int64":
		return int64(1)
	case "string":
		return "hello world"
	case "time.Time":
		return time.Now()
	case "interface{}":
		return 1
	default:
		return 1
	}

}

func LoadTemplate(filename string) (tpl *dbmeta.GenTemplate, err error) {
	baseName := filepath.Base(filename)
	path := filepath.Dir(filename)
	baseTemplates := packr.New("gen", path)
	content, err := baseTemplates.FindString(baseName)
	if err != nil {
		return nil, fmt.Errorf("%s not found internally", baseName)
	}
	tpl = &dbmeta.GenTemplate{Name: "internal://" + filename, Content: content}
	return tpl, nil
}
