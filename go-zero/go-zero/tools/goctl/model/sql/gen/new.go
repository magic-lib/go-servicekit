package gen

import (
	"fmt"

	"github.com/zeromicro/go-zero/tools/goctl/model/sql/template"
	"github.com/zeromicro/go-zero/tools/goctl/util"
	"github.com/zeromicro/go-zero/tools/goctl/util/pathx"
	"github.com/zeromicro/go-zero/tools/goctl/util/stringx"
)

func genNew(table Table, withCache, postgreSql bool) (string, error) {
	text, err := pathx.LoadTemplate(category, modelNewTemplateFile, template.New)
	if err != nil {
		return "", err
	}

	t := fmt.Sprintf(`"%s"`, wrapWithRawString(table.Name.Source(), postgreSql))
	if postgreSql {
		t = "`" + fmt.Sprintf(`"%s"."%s"`, table.Db.Source(), table.Name.Source()) + "`"
	}

	camel := table.Name.ToCamel()
	output, err := util.With("new").
		Parse(text).
		Execute(map[string]any{
			"table":                 t,
			"withCache":             withCache,
			"lowerStartCamelObject": stringx.From(camel).Untitle(),
			"upperStartCamelObject": camel,
			"data":                  table,
		})
	if err != nil {
		return "", err
	}

	return output.String(), nil
}
