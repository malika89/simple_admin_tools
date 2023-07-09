func (l *{{.modelName}}Logic) Get{{.modelName}}List(req *Vo.{{.modelName}}ListReq) (*Vo.{{.modelName}}ListResp, error) {
    var rpcReq := $(SERVICE_LOWER).{{.modelName}}ListReq{}
    if err= mapstructure.Decode(req,&rpcReq);err!=nil {
        return nil,err
    }
	data, err := l.svcCtx.{{.rpcName}}Rpc.Get{{.modelName}}List(l.ctx,&rpcReq)
	if err != nil {
		return nil, err
	}
	resp = &types.{{.modelName}}ListResp{}
	resp.Msg = {{if .useI18n}}l.svcCtx.Trans.Trans(l.ctx, data.Msg){{else}}data.Msg{{end}}
	resp.Data.Total = data.GetTotal()

	for _, v := range data.Data {
		resp.Data.Data = append(resp.Data.Data,
			Vo.{{.modelName}}Info{
				Base{{if .useUUID}}UU{{end}}IDInfo: types.Base{{if .useUUID}}UU{{end}}IDInfo{
					Id:        v.Id,
				},{{.setLogic}}
			})
	}
	return resp, nil
}
