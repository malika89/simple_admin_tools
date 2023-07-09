func (l *{{.modelName}}Logic) Delete{{.modelName}}(req *Vo.Delete{{if .useUUID}}UU{{end}}IDsReq) (*Vo.Delete{{if .useUUID}}UU{{end}Resp, error) {
	data, err := l.svcCtx.{{.rpcName}}Rpc.Delete{{.modelName}}(l.ctx, &$(SERVICE_LOWER).{{if .useUUID}}UU{{end}}IDsReq{
		Ids: req.Ids,
	})
	if err != nil {
		return nil, err
	}

	return &Vo.Delete{{if .useUUID}}UU{{end}Resp{Msg: {{if .useI18n}}l.svcCtx.Trans.Trans(l.ctx, data.Msg){{else}}data.Msg{{end}}}, nil
}
