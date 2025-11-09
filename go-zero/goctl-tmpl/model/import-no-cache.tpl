import (
	"context"
	"database/sql"
	"fmt"
	"github.com/magic-lib/go-plat-utils/utils/httputil"
	"github.com/magic-lib/go-servicekit/tracer"
	"strings"
	{{if .time}}"time"{{end}}

	"gorm.io/gorm"
    "gorm.io/driver/mysql"

    "github.com/magic-lib/go-plat-mysql/sqlstatement"

    {{if .containsPQ}}"github.com/lib/pq"{{end}}
	"github.com/zeromicro/go-zero/core/stores/builder"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/core/stringx"

	{{.third}}
)
