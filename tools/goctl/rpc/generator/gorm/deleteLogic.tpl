func (l *{{.modelName}}Logic) Delete{{.modelName}}(in *{{.projectName}}.{{if .useUUID}}UU{{end}}IDsReq) (*{{.projectName}}.BaseResp, error) {
	record = {{.packageName}}.{{.modelName}}{}
	var strIds string
	for _, v :=range in.Ids {
		strIds +=fmt.Sprintf(",%v",v)
	}
	_, err := l.svcCtx.DB.Where("id in = (?)",strIds).Delete(record).Error
    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &{{.projectName}}.BaseResp{Msg: {{if .useI18n}}i18n.DeleteSuccess{{else}}errormsg.DeleteSuccess {{end}}}, nil
}
