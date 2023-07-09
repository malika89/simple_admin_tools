func (l *{{.logic}}Logic) Get{{.logic}}ById(req *Vo.{{if .useUUID}}UU{{end}}IDReq) (*Vo.{{.modelName}}InfoResp, error) {
	data, err := l.svcCtx.{{.rpcName}}Rpc.Get{{.modelName}}ById(l.ctx, &$(SERVICE_LOWER).{{if .useUUID}}UU{{end}}IDReq{Id: req.Id})
	if err != nil {
		return nil, err
	}

	return &Vo.{{.modelName}}InfoResp{
		BaseDataInfo: Vo.BaseDataInfo{
			Code: 0,
			Msg:  {{if .useI18n}}l.svcCtx.Trans.Trans(l.ctx, data.Msg){{else}}data.Msg{{end}},
		},
		Data: Vo.{{.modelName}}Info{
            Base{{if .useUUID}}UU{{end}}IDInfo: types.Base{{if .useUUID}}UU{{end}}IDInfo{
                Id:        data.Id,
            },{{.setLogic}}
		},
	}, nil
}

