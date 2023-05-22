package {{.packageName}}

import (
	"context"
{{if .hasTime}}     "time"{{end}}

	"{{.projectPath}}/internal/svc"
	"{{.projectPath}}/internal/utils/dberrorhandler"
    "{{.projectPath}}/types/{{.projectName}}"
    "{{.projectPath}}/gorm/{{.packageName}}"

{{if .useI18n}}    "github.com/suyuan32/simple-admin-common/i18n"
{{else}}    "github.com/suyuan32/simple-admin-common/msg/errormsg"
{{end}}{{if .hasUUID}}    "github.com/suyuan32/simple-admin-common/utils/uuidx"
{{end}}
	"github.com/zeromicro/go-zero/core/logx"
)

type {{.modelName}}Logic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func New{{.modelName}}Logic(ctx context.Context, svcCtx *svc.ServiceContext) *{{.modelName}}Logic {
	return &{{.modelName}}Logic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *{{.modelName}}Logic) Create{{.modelName}}(in *{{.projectName}}.{{.modelName}}Info) (*{{.projectName}}.Base{{if .useUUID}}UU{{end}}IDResp, error) {
    record = &{{.packageName}}.{{.modelName}}{
{{.setLogic}}
    }
    err := l.svcCtx.DB.Create(record).Error
	if err != nil {
	    return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}
    return &{{.projectName}}.Base{{if .useUUID}}UU{{end}}IDResp{Id: record.ID{{if .useUUID}}.String(){{end}}, Msg: {{if .useI18n}}i18n.CreateSuccess{{else}}errormsg.CreateSuccess{{end}} }, nil
}