func (l *{{.modelName}}Logic) Update{{.modelName}}(req *Vo.{{.modelName}}Info) (*Vo.BaseMsgResp, error) {
	data, err := l.svcCtx.{{.rpcName}}Rpc.Update{{.modelName}}(l.ctx,
		&$(SERVICE_LOWER).{{.modelName}}Info{
			Id:          req.Id,{{.setLogic}}
		})
	if err != nil {
		return nil, err
	}
	return &Vo.BaseMsgResp{Msg: {{if .useI18n}}l.svcCtx.Trans.Trans(l.ctx, data.Msg){{else}}data.Msg{{end}}}, nil
}
