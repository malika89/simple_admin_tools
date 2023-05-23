package {{.modelPackageName}}

import (
    "database/sql"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	_ = datatypes.JSON{}
)

/*
DB Table Details
-------------------------------------
JSON Sample
-------------------------------------
{{ToJSON .TableInfo.Instance 4}}

*/

 {{if not .Config.AddProtobufAnnotation }}

// {{.StructName}} struct is a row record of the {{.TableName}} table in the {{.DatabaseName}} database
type {{.StructName}} struct {
    {{range .TableInfo.Fields}}{{.}}
    {{end}}
}
{{else}}

// {{.StructName}} struct is a row record of the {{.TableName}} table in the {{.DatabaseName}} database
/*
type {{.StructName}} struct {
    {{range .TableInfo.Fields}}{{.}}
    {{end}}
}
*/

{{end}}


// TableName sets the insert table name for this struct type
func ({{.ShortStructName}} *{{.StructName}}) TableName() string {
	return "{{.TableName}}"
}

// BeforeSave invoked before saving, return an error if field is not populated.
func ({{.ShortStructName}} *{{.StructName}}) BeforeSave(tx *gorm.DB) error {
	return nil
}

// Prepare invoked before saving, can be used to populate fields etc.
func ({{.ShortStructName}} *{{.StructName}}) Prepare() {
}
