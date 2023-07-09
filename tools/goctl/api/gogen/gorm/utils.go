package gorm

import (
	"bytes"
	"fmt"
	"github.com/gookit/color"
	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/tools/goctl/api/spec"
	"github.com/zeromicro/go-zero/tools/goctl/api/util"
	apiutil "github.com/zeromicro/go-zero/tools/goctl/api/util"
	"github.com/zeromicro/go-zero/tools/goctl/pkg/golang"
	"github.com/zeromicro/go-zero/tools/goctl/util/pathx"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"
)

type fileGenConfig struct {
	dir             string
	subdir          string
	filename        string
	templateName    string
	category        string
	templateFile    string
	builtinTemplate string
	data            any
}

func MaybeCreateFile(dir, subdir, file string) (fp *os.File, created bool, err error) {
	logx.Must(pathx.MkdirIfNotExist(path.Join(dir, subdir)))
	fpath := path.Join(dir, subdir, file)
	if pathx.FileExists(fpath) {
		os.Remove(fpath)
		logx.Debug("%s exists, rewriting generation\n", fpath)
	}

	fp, err = pathx.CreateIfNotExist(fpath)
	created = err == nil
	return
}

func genFile(c fileGenConfig) error {
	fp, created, err := MaybeCreateFile(c.dir, c.subdir, c.filename)
	if err != nil {
		return err
	}
	if !created {
		return nil
	}
	defer func(fp *os.File) {
		_ = fp.Close()
	}(fp)

	var text string
	if len(c.category) == 0 || len(c.templateFile) == 0 {
		text = c.builtinTemplate
	} else {
		text, err = pathx.LoadTemplate(c.category, c.templateFile, c.builtinTemplate)
		if err != nil {
			return err
		}
	}

	t := template.Must(template.New(c.templateName).Parse(text))
	buffer := new(bytes.Buffer)
	err = t.Execute(buffer, c.data)
	if err != nil {
		return err
	}

	code := golang.FormatCode(buffer.String())
	_, err = fp.WriteString(code)
	return err
}

func getMiddleware(api *spec.ApiSpec) []string {
	result := collection.NewSet()
	for _, g := range api.Service.Groups {
		middleware := g.GetAnnotation("middleware")
		if len(middleware) > 0 {
			for _, item := range strings.Split(middleware, ",") {
				result.Add(strings.TrimSpace(item))
			}
		}
	}

	return result.KeysStr()
}

func responseGoTypeName(r spec.Route, pkg ...string) string {
	if r.ResponseType == nil {
		return ""
	}

	resp := golangExpr(r.ResponseType, pkg...)
	switch r.ResponseType.(type) {
	case spec.DefineStruct:
		if !strings.HasPrefix(resp, "*") {
			return "*" + resp
		}
	}

	return resp
}

func golangExpr(ty spec.Type, pkg ...string) string {
	switch v := ty.(type) {
	case spec.PrimitiveType:
		return v.RawName
	case spec.DefineStruct:
		if len(pkg) > 1 {
			panic("package cannot be more than 1")
		}

		if len(pkg) == 0 {
			return v.RawName
		}

		return fmt.Sprintf("%s.%s", pkg[0], cases.Title(language.English, cases.NoLower).String(v.RawName))
	case spec.ArrayType:
		if len(pkg) > 1 {
			panic("package cannot be more than 1")
		}

		if len(pkg) == 0 {
			return v.RawName
		}

		return fmt.Sprintf("[]%s", golangExpr(v.Value, pkg...))
	case spec.MapType:
		if len(pkg) > 1 {
			panic("package cannot be more than 1")
		}

		if len(pkg) == 0 {
			return v.RawName
		}

		return fmt.Sprintf("map[%s]%s", v.Key, golangExpr(v.Value, pkg...))
	case spec.PointerType:
		if len(pkg) > 1 {
			panic("package cannot be more than 1")
		}

		if len(pkg) == 0 {
			return v.RawName
		}

		return fmt.Sprintf("*%s", golangExpr(v.Type, pkg...))
	case spec.InterfaceType:
		return v.RawName
	}

	return ""
}

func requestGoTypeName(r spec.Route, pkg ...string) string {
	if r.RequestType == nil {
		return ""
	}

	return golangExpr(r.RequestType, pkg...)
}

func backupAndSweep(apiFile string) error {
	var err error
	var wg sync.WaitGroup

	wg.Add(2)
	_ = os.MkdirAll(tmpDir, os.ModePerm)

	go func() {
		_, fileName := filepath.Split(apiFile)
		_, e := apiutil.Copy(apiFile, fmt.Sprintf(path.Join(tmpDir, tmpFile), fileName, time.Now().Unix()))
		if e != nil {
			err = e
		}
		wg.Done()
	}()
	go func() {
		if e := sweep(); e != nil {
			err = e
		}
		wg.Done()
	}()
	wg.Wait()

	return err
}

func sweep() error {
	keepTime := time.Now().AddDate(0, 0, -7)
	return filepath.Walk(tmpDir, func(fpath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		pos := strings.LastIndexByte(info.Name(), '-')
		if pos > 0 {
			timestamp := info.Name()[pos+1:]
			seconds, err := strconv.ParseInt(timestamp, 10, 64)
			if err != nil {
				// print error and ignore
				fmt.Println(color.Red.Sprintf("sweep ignored file: %s", fpath))
				return nil
			}

			tm := time.Unix(seconds, 0)
			if tm.Before(keepTime) {
				if err := os.RemoveAll(fpath); err != nil {
					fmt.Println(color.Red.Sprintf("failed to remove file: %s", fpath))
					return err
				}
			}
		}

		return nil
	})
}

func getAllFile(dirPath string) ([]string, error) {
	var s []string
	rd, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return s, err
	}
	for _, fi := range rd {
		if !fi.IsDir() {
			fullName := filepath.Join(dirPath, fi.Name())
			s = append(s, fullName)
		}
	}
	return s, nil
}

func writeProperty(writer io.Writer, name, tag, comment string, tp spec.Type, doc spec.Doc, indent int) error {
	// write doc for swagger
	var err error
	hasValidator := false

	for _, v := range doc {
		_, _ = fmt.Fprintf(writer, "\t%s\n", v)
		if util.HasCustomValidation(v) {
			hasValidator = true
		}
	}

	if !hasValidator {
		validatorComment, err := util.ConvertValidateTagToSwagger(tag)
		if err != nil {
			return err
		}

		for _, v := range validatorComment {
			_, _ = fmt.Fprint(writer, v)
		}
	}

	util.WriteIndent(writer, indent)

	if len(comment) > 0 {
		comment = strings.TrimPrefix(comment, "//")
		comment = "//" + comment
		_, err = fmt.Fprintf(writer, "%s %s %s %s\n", cases.Title(language.English, cases.NoLower).String(name), tp.Name(), tag, comment)
	} else {
		_, err = fmt.Fprintf(writer, "%s %s %s\n", cases.Title(language.English, cases.NoLower).String(name), tp.Name(), tag)
	}

	return err
}

func ConvertRoutePathToSwagger(data string) string {
	splitData := strings.Split(data, "/")
	for i, v := range splitData {
		if strings.Contains(v, ":") {
			splitData[i] = "{" + v[1:] + "}"
		}
	}
	return strings.Join(splitData, "/")
}
