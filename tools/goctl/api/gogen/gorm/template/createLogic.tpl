package {{.modelNameLowerCase}}

import (
	"context"
	"github.com/mitchellh/mapstructure"

	"{{.projectPackage}}/internal/svc"
	"{{.projectPackage}}/internal/types"
	"{{.rpcPackage}}"

	"github.com/zeromicro/go-zero/core/logx"
)

type {{.modelName}}Logic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func New{{.modelName}}Logic(ctx context.Context, svcCtx *svc.ServiceContext) *{{.modelName}}Logic {
	return &{{.modelName}}Logic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *{{.modelName}}Logic) Create{{.modelName}}(req *Vo.Create{{.modelName}}Req) (resp *Vo.Create{{.modelName}}Resp, err error) {
	data, err := l.svcCtx.{{.rpcName}}Rpc.Create{{.modelName}}(l.ctx,
		&$(SERVICE_LOWER).{{.modelName}}Info{ {{.setLogic}}
		})
	if err != nil {
		return nil, err
	}
	return &Vo.Create{{.modelName}}Resp{Msg: {{if .useI18n}}l.svcCtx.Trans.Trans(l.ctx, data.Msg){{else}}data.Msg{{end}}}, nil
}
