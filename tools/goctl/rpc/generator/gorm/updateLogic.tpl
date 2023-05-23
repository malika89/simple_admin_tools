func (l *{{.modelName}}Logic) Update{{.modelName}}(in *{{.projectName}}.{{.modelName}}Info) (*{{.projectName}}.BaseResp, error) {
    record := &{{.packageName}}.{{.modelName}}{
{{.setLogic}}
       }
    err := l.svcCtx.DB.Updates(record).Error
    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &{{.projectName}}.BaseResp{Msg: {{if .useI18n}}i18n.CreateSuccess{{else}}errormsg.UpdateSuccess{{end}} }, nil
}
