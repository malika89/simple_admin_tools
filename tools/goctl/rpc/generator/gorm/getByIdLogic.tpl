func (l *{{.modelName}}Logic) Get{{.modelName}}ById(in *{{.projectName}}.{{if .useUUID}}UU{{end}}IDReq) (*{{.projectName}}.{{.modelName}}Info, error) {
	record := &{{.packageName}}.{{.modelName}}{}
	err := l.svcCtx.DB.Where("id = ?",{{if .useUUID}}uuidx.ParseUUIDString({{end}}in.Id{{if .useUUID}}){{end}}).Take(&record).Error
	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	return &{{.projectName}}.{{.modelName}}Info{
		Id:          record.ID,
		CreatedAt:   result.CreatedAt.UnixMilli(),
		UpdatedAt:   result.UpdatedAt.UnixMilli(),
{{.listData}}
	}, nil
}

