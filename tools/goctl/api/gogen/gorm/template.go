package gorm

import (
	_ "embed"
)

var (
	//go:embed template/createLogic.tpl
	createTpl string

	//go:embed template/updateLogic.tpl
	updateTpl string

	//go:embed template/getListLogic.tpl
	getListLogicTpl string

	//go:embed template/getByIdLogic.tpl
	getByIdLogicTpl string

	//go:embed template/deleteLogic.tpl
	deleteLogicTpl string

	//go:embed template/api.tpl
	apiTpl string

	//go:embed template/Vo.go.tpl
	voTpl string

	//go:embed template/basic.tpl
	basicTpl string
)

const (
	category                   = "api"
	contextTemplateFile        = "context.tpl"
	handlerTemplateFile        = "handler.tpl"
	logicTemplateFile          = "logic.tpl"
	routesTemplateFile         = "routes.tpl"
	typesTemplateFile          = "types.tpl"
	dbErrorHandlerTemplateFile = "dberrorhandler.tpl"
	routesAdditionTemplateFile = "route-addition.tpl"
)

var templates = map[string]string{
	contextTemplateFile: contextTemplate,
	handlerTemplateFile: handlerTemplate,
	logicTemplateFile:   logicTemplate,
	routesTemplateFile:  routesTemplate,
	typesTemplateFile:   typesTemplate,
}
