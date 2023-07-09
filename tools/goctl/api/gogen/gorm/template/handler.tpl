package {{.PkgName}}

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	{{.ImportPackages}}
)

{{range $i, $handler := .Handlers}}
{{ $handler.HandlerDoc }}
func {{ $handler.HandlerName}}(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		{{if $handler.HasRequest}}var req types.{{ $handler.RequestType }}
		if err := httpx.Parse(r, &req, true); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		{{end}}
		l := {{ $handler.LogicName }}.New{{ $handler.LogicType }}(r.Context(), svcCtx)
		{{if $handler.HasResp }}resp, {{end}}
		err := l.{{ $handler.Call }}({{if $handler.HasRequest}}&req{{end}})
		if err != nil {
		    {{if $handler.TransErr}}err = svcCtx.Trans.TransError(r.Context(), err){{end}}
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			{{if $handler.HasResp }}httpx.OkJsonCtx(r.Context(), w, resp){{else}}httpx.Ok(w){{end}}
		}
	}
}
{{end}}


