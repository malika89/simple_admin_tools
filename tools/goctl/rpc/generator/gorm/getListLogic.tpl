func (l *{{.modelName}}Logic) Get{{.modelName}}List(in *{{.projectName}}.{{.modelName}}ListReq) (*{{.projectName}}.{{.modelName}}ListResp, error) {
    var (
    	lists []{{.packageName}}.{{.modelName}}
    	count int64
    )

	resp := &{{.projectName}}.{{.modelName}}ListResp{}
	db := l.svcCtx.DB
	for k, v:= range in.Filters {
		db = db.Where("? = ?",k, v)
	}
	db.Count(&count)
    err := db.Limit(int(in.PageSize)).Offset(int((in.Page - 1) * in.PageSize)).Find(&lists).Error

    if err != nil {
    	return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
    }
    resp.Total = uint64(count)
	resp.Total = result.PageDetails.Total

	for _, v := range result.List {
		resp.Data = append(resp.Data, &{{.projectName}}.{{.modelName}}Info{
			Id:          v.ID{{if .useUUID}}.String(){{end}},
			CreatedAt:   v.CreatedAt.UnixMilli(),
			UpdatedAt:   v.UpdatedAt.UnixMilli(),
{{.listData}}
		})
	}

	return resp, nil
}
