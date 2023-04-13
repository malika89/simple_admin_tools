package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zeromicro/go-zero/tools/goctl/extra/ent/template"
	"github.com/zeromicro/go-zero/tools/goctl/rpc/execx"
	proto2 "github.com/zeromicro/go-zero/tools/goctl/rpc/generator/proto"
	"github.com/zeromicro/go-zero/tools/goctl/rpc/parser"
	"github.com/zeromicro/go-zero/tools/goctl/util/console"
	"github.com/zeromicro/go-zero/tools/goctl/util/ctx"
	"github.com/zeromicro/go-zero/tools/goctl/util/pathx"
)

type ZRpcContext struct {
	// Sre is the source file of the proto.
	Src string
	// ProtoCmd is the command to generate proto files.
	ProtocCmd string
	// ProtoGenGrpcDir is the directory to store the generated proto files.
	ProtoGenGrpcDir string
	// ProtoGenGoDir is the directory to store the generated go files.
	ProtoGenGoDir string
	// IsGooglePlugin is the flag to indicate whether the proto file is generated by google plugin.
	IsGooglePlugin bool
	// GoOutput is the output directory of the generated go files.
	GoOutput string
	// GrpcOutput is the output directory of the generated grpc files.
	GrpcOutput string
	// Output is the output directory of the generated files.
	Output string
	// Multiple is the flag to indicate whether the proto file is generated in multiple mode.
	Multiple bool
	// Schema is the ent schema path
	Schema string
	// Ent
	Ent bool
	// ModuleName is the module name in go mod
	ModuleName string
	// GoZeroVersion describes the version of Go Zero
	GoZeroVersion string
	// ToolVersion describes the version of Simple Admin Tools
	ToolVersion string
	// Port describes the service port exposed
	Port int
	// MakeFile describes whether generate makefile
	MakeFile bool
	// DockerFile describes whether generate dockerfile
	DockerFile bool
	// Gitlab describes whether to use gitlab-ci
	Gitlab bool
	// DescDir describes whether to create desc folder for splitting proto files
	UseDescDir bool
	// RpcName describes the rpc name when create new project
	RpcName string
}

// Generate generates a rpc service, through the proto file,
// code storage directory, and proto import parameters to control
// the source file and target location of the rpc service that needs to be generated
func (g *Generator) Generate(zctx *ZRpcContext) error {
	console.NewColorConsole(true).Info(aurora.Green("Generating...").String())

	abs, err := filepath.Abs(zctx.Output)
	if err != nil {
		return err
	}

	err = pathx.MkdirIfNotExist(abs)
	if err != nil {
		return err
	}

	// merge proto files
	protoDir := filepath.Join(abs, "desc")

	if pathx.Exists(protoDir) {
		protoFileAbsPath, err := filepath.Abs(zctx.Src)
		if err != nil {
			return err
		}

		if err = proto2.MergeProto(&proto2.ProtoContext{
			ProtoDir:   protoDir,
			OutputPath: protoFileAbsPath,
		}); err != nil {
			return err
		}
	}

	err = g.Prepare()
	if err != nil {
		return err
	}

	if zctx.ModuleName != "" {
		_, err = execx.Run("go mod init "+zctx.ModuleName, abs)
		if err != nil {
			return err
		}
	}

	if zctx.GoZeroVersion != "" && zctx.ToolVersion != "" {
		_, err := execx.Run(fmt.Sprintf("goctls migrate --zero_version %s --tool_version %s", zctx.GoZeroVersion, zctx.ToolVersion),
			abs)
		if err != nil {
			return err
		}
	}

	projectCtx, err := ctx.Prepare(abs)
	if err != nil {
		return err
	}

	p := parser.NewDefaultProtoParser()
	proto, err := p.Parse(zctx.Src, zctx.Multiple)
	if err != nil {
		return err
	}

	dirCtx, err := mkdir(projectCtx, proto, g.cfg, zctx)
	if err != nil {
		return err
	}

	err = g.GenEtc(dirCtx, proto, g.cfg, zctx)
	if err != nil {
		return err
	}

	err = g.GenPb(dirCtx, zctx)
	if err != nil {
		return err
	}

	err = g.GenConfig(dirCtx, proto, g.cfg, zctx)
	if err != nil {
		return err
	}

	err = g.GenSvc(dirCtx, proto, g.cfg, zctx)
	if err != nil {
		return err
	}

	err = g.GenLogic(dirCtx, proto, g.cfg, zctx)
	if err != nil {
		return err
	}

	err = g.GenServer(dirCtx, proto, g.cfg, zctx)
	if err != nil {
		return err
	}

	err = g.GenMain(dirCtx, proto, g.cfg, zctx)
	if err != nil {
		return err
	}

	err = g.GenCall(dirCtx, proto, g.cfg, zctx)

	if zctx.MakeFile {
		err = g.GenMakefile(dirCtx, proto, g.cfg, zctx)
		if err != nil {
			return err
		}
	}

	if zctx.DockerFile {
		err = g.GenDockerfile(dirCtx, proto, g.cfg, zctx)
		if err != nil {
			return err
		}
	}

	if zctx.Gitlab {
		err = g.GenGitlab(dirCtx, proto, g.cfg, zctx)
		if err != nil {
			return err
		}
	}

	if zctx.UseDescDir {
		err = g.GenBaseDesc(dirCtx, proto, g.cfg, zctx)
		if err != nil {
			return err
		}
	}

	// generate ent
	if zctx.Ent {
		_, err := execx.Run(fmt.Sprintf("go run -mod=mod entgo.io/ent/cmd/ent new %s",
			dirCtx.GetServiceName().ToCamel()), abs)
		if err != nil {
			return err
		}

		_, err = execx.Run("go mod tidy", abs)
		if err != nil {
			return err
		}

		_, err = execx.Run("go run -mod=mod entgo.io/ent/cmd/ent generate ./ent/schema", abs)
		if err != nil {
			return err
		}

		err = pathx.MkdirIfNotExist(filepath.Join(abs, "ent", "template"))
		if err != nil {
			return err
		}

		paginationTplPath := filepath.Join(abs, "ent", "template", "pagination.tmpl")
		notEmptyTplPath := filepath.Join(abs, "ent", "template", "not_empty_update.tmpl")
		if !pathx.FileExists(paginationTplPath) {
			err = os.WriteFile(paginationTplPath, []byte(template.PaginationTmpl), os.ModePerm)
			if err != nil {
				return err
			}

			err = os.WriteFile(notEmptyTplPath, []byte(template.NotEmptyTmpl), os.ModePerm)
			if err != nil {
				return err
			}
		}

		// gen ent error handler
		err = g.GenErrorHandler(dirCtx, proto, g.cfg, zctx)
		if err != nil {
			return err
		}

		// gen ent transaction util
		err = g.GenEntTx(dirCtx, proto, g.cfg, zctx)
		if err != nil {
			return err
		}

		_, err = execx.Run("go mod tidy", abs)
		if err != nil {
			return err
		}
	}

	console.NewColorConsole().MarkDone()

	return err
}
