package {{.pkgName}}

import (
	{{.imports}}
)

type {{.logic}} struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func New{{.logic}}(ctx context.Context, svcCtx *svc.ServiceContext) *{{.logic}} {
	return &{{.logic}}{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}
{{range $i, $lo := .Logics}}
{{ if eq $lo.Method "Create"}}
//新增
func (l *{{ $.logic }}) {{$lo.Function}}({{$lo.Request}}) {{$lo.ResponseType}} {
    var (
        rpcReq = {{ $.rpcPackageLower}}.{{ $.logic }}Info{}
    )
    if err= mapstructure.Decode(req,&rpcReq);err!=nil {
    	return
    }
	data, err := l.svcCtx.{{ $.rpcName }}.Create{{ $.logic }}(l.ctx,&rpcReq)
	if err != nil {
		return
	}
	resp.Msg =  {{if $.useI18n}}l.svcCtx.Trans.Trans(l.ctx, data.Msg){{else}}data.Msg{{end}}
	return
}
{{else if eq $lo.Method "Update"}}
//修改
func (l *{{ $.logic}}) {{$lo.Function}}({{$lo.Request}}) {{$lo.ResponseType}} {
	data, err := l.svcCtx.{{ $.rpcName }}.Update{{ $.logic }}(l.ctx,
		&{{ $.rpcPackageLower }}.{{ $.logic }}Info{
			Id:          req.Id,{{$lo.SetLogic}}
		})
	if err != nil {
		return
	}
	resp.Msg =  {{if $.useI18n}}l.svcCtx.Trans.Trans(l.ctx, data.Msg){{else}}data.Msg{{end}}
	return
}
{{else if eq $lo.Method "GetById"}}
//详情
func (l *{{ $.logic}}) {{$lo.Function}}({{$lo.Request}}) {{$lo.ResponseType}} {
	data, err := l.svcCtx.{{ $.rpcName }}.Get{{ $.logic }}ById(l.ctx, &{{ $.rpcPackageLower}}.{{if $.useUUID}}UU{{end}}IDReq{Id: req.Id})
	if err != nil {
		return
	}
	resp.BaseDataInfo=types.BaseDataInfo{
                Data:        data,
            }
	return
}
{{else if eq $lo.Method "GetList"}}
//列表
func (l *{{ $.logic}}) {{$lo.Function}}({{$lo.Request}}) {{$lo.ResponseType}} {
    var (
        rpcReq = {{ $.rpcPackageLower}}.{{ $.logic }}ListReq{}
    )
    if err= mapstructure.Decode(req,&rpcReq);err!=nil {
        return
    }
	data, err := l.svcCtx.{{ $.rpcName }}.Get{{ $.logic }}List(l.ctx,&rpcReq)
	if err != nil {
		return
	}
	resp.Msg = {{if $.useI18n}}l.svcCtx.Trans.Trans(l.ctx, data.Msg){{else}}data.Msg{{end}}
	resp.Data.Total = data.GetTotal()

	for _, v := range data.Data {
		resp.Data.Data = append(resp.Data.Data,
			types.{{ $.logic }}Info{
				Base{{if $.useUUID}}UU{{end}}IDInfo: types.Base{{if $.useUUID}}UU{{end}}IDInfo{
					Id:        v.Id,
				},{{$lo.SetLogic}}
			})
	}
	return resp, nil
}
{{else if eq $lo.Method "Delete"}}
//删除
func (l *{{ $.logic }}) {{$lo.Function}}({{$lo.Request}}) {{$lo.ResponseType}} {
	data, err := l.svcCtx.{{ $.rpcName }}.Delete{{ $.logic }}(l.ctx, &{{ $.rpcPackageLower}}.{{if $.useUUID}}UU{{end}}IDsReq{
		Ids: req.Ids,
	})
	if err != nil {
		return
	}
	resp.Msg = {{if $.useI18n}}l.svcCtx.Trans.Trans(l.ctx, data.Msg){{else}}data.Msg{{end}}
	return
}
{{else}}
func (l *{{ $.logic}}) {{$lo.Function}}({{$lo.Request}}) {{$lo.ResponseType}} {
	// todo: add your logic here and delete this line
	{{$lo.ReturnString}}
}
{{end}}
{{end}}
