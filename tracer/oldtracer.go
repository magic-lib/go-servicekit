package tracer

//import (
//"context"
//"contrib.go.opencensus.io/exporter/ocagent"
//"git.code.oa.com/odp-go/go-tools/utils"
//"git.code.oa.com/odp-go/go-tools/utils/database"
//"go.opencensus.io/plugin/ochttp/propagation/b3"
//"go.opencensus.io/trace"
//"log"
//"net"
//"net/http"
//"sync"
//"time"
//)

//var (
//	b3Format              = b3.HTTPFormat{}
//	muxHeader             = &sync.Mutex{}
//	defaultAddress string = "localhost:55678"
//)
//

//
//// startSpan 整个开始的span，从外部传入
//func startSpan(req *http.Request, name string) (context.Context, *trace.Span) {
//	var oldCtx context.Context
//	if req == nil {
//		oldCtx = context.Background()
//	} else {
//		oldCtx = req.Context()
//	}
//
//	var span *trace.Span
//	var ctx context.Context
//	contextWithSpan, ok := spanContextFromRequest(req)
//	if !ok {
//		ctx, span = trace.StartSpan(
//			oldCtx,
//			name,
//			trace.WithSpanKind(trace.SpanKindServer),
//		)
//
//		parentSpan := trace.FromContext(oldCtx)
//		if parentSpan != nil {
//			isParentExist := parentSpan.SpanContext() != trace.SpanContext{}
//			if isParentExist {
//				// 进行link链接
//				span.AddLink(trace.Link{
//					TraceID: parentSpan.SpanContext().TraceID,
//					SpanID:  parentSpan.SpanContext().SpanID,
//					Type:    trace.LinkTypeChild,
//				})
//			}
//		}
//	} else {
//		ctx, span = trace.StartSpanWithRemoteParent(
//			oldCtx,
//			name,
//			contextWithSpan,
//			trace.WithSpanKind(trace.SpanKindServer),
//		)
//		span.AddLink(trace.Link{
//			TraceID: contextWithSpan.TraceID,
//			SpanID:  contextWithSpan.SpanID,
//			Type:    trace.LinkTypeChild,
//		})
//	}
//
//	return ctx, span
//}
//
//// newSpan 新建span，内部之间调用
//func newSpan(req *http.Request, name string) (context.Context, *trace.Span) {
//	var oldCtx context.Context
//	if req == nil {
//		oldCtx = context.Background()
//	} else {
//		oldCtx = req.Context()
//	}
//
//	var span *trace.Span
//	var ctx context.Context
//
//	ctx, span = trace.StartSpan(
//		oldCtx,
//		name,
//		trace.WithSpanKind(trace.SpanKindServer),
//	)
//
//	parentSpan := trace.FromContext(oldCtx)
//	if parentSpan != nil {
//		isParentExist := parentSpan.SpanContext() != trace.SpanContext{}
//		if isParentExist {
//			// 进行link链接
//			span.AddLink(trace.Link{
//				TraceID: parentSpan.SpanContext().TraceID,
//				SpanID:  parentSpan.SpanContext().SpanID,
//				Type:    trace.LinkTypeChild,
//			})
//		}
//	}
//
//	return ctx, span
//}
//
//// NewSpan 新建立一个span
//func NewSpan(req *http.Request, name string, isStart bool) (context.Context, *trace.Span) {
//	var span *trace.Span
//	var ctx context.Context
//	if isStart {
//		ctx, span = startSpan(req, name)
//	} else {
//		ctx, span = newSpan(req, name)
//	}
//	return ctx, span
//}
//
//// StartNewSpan 自带关闭方法
//func StartNewSpan(req *http.Request, name string, isStart bool, callback func(ctx context.Context, spans *trace.Span)) {
//	var span *trace.Span
//	var ctx context.Context
//	ctx, span = NewSpan(req, name, isStart)
//	if span == nil {
//		callback(ctx, nil)
//		return
//	}
//	defer span.End()
//	callback(ctx, span)
//	return
//}
//
////
////func SetTraceId(req *http.Request, span *trace.Span, traceId string) bool {
////	if span == nil {
////		return false
////	}
////
////	traceID, retTrue := b3.ParseTraceID(traceId)
////	if !retTrue {
////		return false
////	}
////
////	spanContext := span.SpanContext()
////	spanContext.TraceID = traceID
////
////	b3Format.SpanContextToRequest(spanContext, req)
////
////	return true
////}
////
////func SetSpanId(req *http.Request, span *trace.Span, spanId string) bool {
////	if span == nil {
////		return false
////	}
////
////	spanID, retTrue := b3.ParseSpanID(spanId)
////	if !retTrue {
////		return false
////	}
////
////	spanContext := span.SpanContext()
////	spanContext.SpanID = spanID
////
////	b3Format.SpanContextToRequest(spanContext, req)
////
////	return true
////}
//// NewSpanById 通过traceId创建新Span
//func NewSpanById(req *http.Request, name string, traceId, spanId string) (context.Context, *trace.Span) {
//	if traceId == "" || req == nil || traceId == "" || spanId == "" {
//		return nil, nil
//	}
//
//	traceID, retTrue := b3.ParseTraceID(traceId)
//	if !retTrue {
//		return nil, nil
//	}
//	spanID, retTrue := b3.ParseSpanID(spanId)
//	if !retTrue {
//		return nil, nil
//	}
//
//	_, parentSpan := trace.StartSpan(req.Context(), name)
//	pctx := parentSpan.SpanContext()
//	pctx.TraceID = traceID
//	pctx.SpanID = spanID
//
//	ctx := NewSpanContext(req.Context(), parentSpan)
//	newReq := req.WithContext(ctx)
//
//	return NewSpan(newReq, name, false)
//}
//
//// spanContextFromRequest 从request获取spanContext
//func spanContextFromRequest(req *http.Request) (trace.SpanContext, bool) {
//	if req == nil {
//		return trace.SpanContext{}, false
//	}
//	return b3Format.SpanContextFromRequest(req)
//}
//
//// SpanContextToRequest 将spanContext设置到request
//func SpanContextToRequest(req *http.Request, sc trace.SpanContext) {
//	if req == nil {
//		return
//	}
//	muxHeader.Lock()
//	//header中设置
//	b3Format.SpanContextToRequest(sc, req)
//	muxHeader.Unlock()
//}
//
//// NewSpanContext 设置parentContext，供下个span使用
//func NewSpanContext(current context.Context, s *trace.Span) context.Context {
//	return trace.NewContext(current, s)
//}
