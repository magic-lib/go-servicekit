package tracer_test

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/magic-lib/go-plat-utils/cond"
	"github.com/magic-lib/go-servicekit/tracer"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"net/http"
	"testing"
	"time"
)

var tc = tracer.TraceConfig{
	Namespace:   "my-service",
	ServiceName: "my-http-service",
	//Endpoint:    "192.168.2.84:4318",
	//Endpoint: "192.168.2.84:14268",
	Endpoint: "http://202.xxxxxx:14268/api/traces",
}

func initTraceProvider() *sdktrace.TracerProvider {
	tracerProvider, err := tc.InitTrace()
	if err != nil {
		fmt.Println(err, "Failed to initialize tracer provider", tc.ServiceName)
		return nil
	}
	//defer func() {
	//	tc.Stop()
	//}()

	return tracerProvider
}

// TestHttpTrace  Tracer Provider。
func TestHttpTrace(t *testing.T) {

	initTraceProvider()

	http.Handle("/mmmm/bbbb", tc.HttpMiddleware(nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, span := tc.StartSpan(r.Context(), "abc")
		defer span.End()

		tracer.SetTags(span, map[string]any{
			"name": "aa",
			"age":  12,
		})

		_, span2 := tc.StartSpan(r.Context(), "bb")
		defer span2.End()

		tracer.SetErrorTag(span2, fmt.Errorf("no hello World"))

		_, _ = fmt.Fprintf(w, "Hello, World!")
	})))

	fmt.Print(http.ListenAndServe(":8080", nil))
}

func TestGinTrace(t *testing.T) {
	initTraceProvider()

	g := gin.Default()
	g.Use(tc.GinMiddleware())
	g.Handle(http.MethodGet, "/gin/:name", func(c *gin.Context) {
		kkk := c.FullPath()
		fmt.Println(kkk)

		_, span := tc.StartSpan(c.Request.Context(), "gin-aaa")
		defer span.End()

		_, _ = fmt.Fprintf(c.Writer, "Gin Hello, World!")

	})
	g.Handle(http.MethodGet, "/gin/:name/aa/:id", func(c *gin.Context) {
		_, span := tc.StartSpan(c.Request.Context(), "gin-aaa-22")
		defer span.End()

		_, _ = fmt.Fprintf(c.Writer, "Gin22 Hello, World!")

	})
	g.Run(":8081")
}

type CompanyInfo struct {
	Id                         int64          `db:"id" json:"id"`
	CompanyNo                  string         `db:"company_no" json:"company_no"`                                     // 企业编号
	CompanyName                string         `db:"company_name" json:"company_name"`                                 // 企业名称
	CreditAffordabilityFormula string         `db:"credit_affordability_formula" json:"credit_affordability_formula"` // 企业员工信审净收入计算公式
	CreditTagFormula           string         `db:"credit_tag_formula" json:"credit_tag_formula"`                     // 企业员工信审可贷标签公式
	MouSalaryDay               int            `db:"mou_salary_day" json:"mou_salary_day"`                             // 企业mou中的工资日
	Extend                     sql.NullString `db:"extend" json:"extend"`                                             // 扩展字段
	IsDisabled                 uint8          `db:"is_disabled" json:"is_disabled"`                                   // 是否禁用
}

func TestGormTrace(t *testing.T) {
	initTraceProvider()

	dsn := "root:xxxxx@tcp(xxxxxxx:3306)/zamloan2_member?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return
	}

	oldCtx := context.Background()

	db = tc.GormMiddleware(oldCtx, db)

	var retrievedUser CompanyInfo
	result := db.Table("company_info").Find(&retrievedUser)
	if result.Error != nil {
		fmt.Println("failed to retrieve user:", result.Error)
		return
	}
	fmt.Printf("retrieved user: %+v\n", retrievedUser)

	time.Sleep(time.Second * 5)

}
func TestGoZeroTrace(t *testing.T) {
	initTraceProvider()

	oneServer := rest.MustNewServer(rest.RestConf{
		ServiceConf: service.ServiceConf{
			Name: tc.ServiceName,
		},
		Host: "0.0.0.0",
		Port: 8082,
	}, rest.WithCorsHeaders())

	oneServer.AddRoutes(
		[]rest.Route{
			{
				Method: http.MethodGet,
				Path:   "/member/:member_id",
				Handler: func(writer http.ResponseWriter, r *http.Request) {

					_, span := tc.StartSpan(r.Context(), "zero-aaa")
					defer span.End()

					_, _ = writer.Write([]byte("go-zero http"))
				},
			},
		},
	)

	oneServer.Use(tc.GoZeroMiddleware(oneServer))

	oneServer.Start()
}
func TestIsNil(t *testing.T) {
	var provider trace.TracerProvider = otel.GetTracerProvider()
	if cond.IsNil(provider) {
		fmt.Println("nil")
		return
	}
	dd := provider.Tracer("aaaa")
	fmt.Println("not nil", dd)
}
