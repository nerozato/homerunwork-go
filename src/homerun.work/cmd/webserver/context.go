package main

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
)

//type to use for context keys
type ctxKey struct {
	name string
}

//String : string value for a context key
func (k *ctxKey) String() string {
	return k.name
}

//context keys
var (
	ctxKeyBookID          = &ctxKey{"BookId"}
	ctxKeyErr             = &ctxKey{"Error"}
	ctxKeyCustomHost      = &ctxKey{"CustomHost"}
	ctxKeyIsSignUp        = &ctxKey{"IsSignUp"}
	ctxKeyLogger          = &ctxKey{"Logger"}
	ctxKeyMsg             = &ctxKey{"Message"}
	ctxKeyPaymentID       = &ctxKey{"PaymentId"}
	ctxKeyProviderURLName = &ctxKey{"ProviderUrlName"}
	ctxKeyRequestID       = &ctxKey{"RequestId"}
	ctxKeyServiceID       = &ctxKey{"ServiceId"}
	ctxKeyStats           = &ctxKey{"Stats"}
	ctxKeyTitleAlert      = &ctxKey{"TitleNotification"}
	ctxKeyURLShort        = &ctxKey{"UrlShort"}
	ctxKeyUserID          = &ctxKey{"UserId"}
	ctxKeyTimeZone        = &ctxKey{"TimeZone"}
	ctxKeyType            = &ctxKey{"Type"}
)

//GetCtxBookID : retrieve the book ID from the context
func GetCtxBookID(ctx context.Context) string {
	v, ok := ctx.Value(ctxKeyBookID).(string)
	if !ok || v == "" {
		return ""
	}
	return v
}

//SetCtxBookID : set the book ID in the context
func SetCtxBookID(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, ctxKeyBookID, v)
}

//GetCtxCustomHost : retrieve thehost from the context
func GetCtxCustomHost(ctx context.Context) string {
	v, ok := ctx.Value(ctxKeyCustomHost).(string)
	if !ok || v == "" {
		return ""
	}
	return v
}

//SetCtxCustomHost : set the host in the context
func SetCtxCustomHost(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, ctxKeyCustomHost, v)
}

//GetCtxErr : retrieve the error from the context
func GetCtxErr(ctx context.Context) string {
	v, ok := ctx.Value(ctxKeyErr).(string)
	if !ok || v == "" {
		return ""
	}
	return v
}

//SetCtxErr : set the error in the context
func SetCtxErr(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, ctxKeyErr, v)
}

//GetCtxIsSignUp : retrieve the flag indicating sign-up from the context
func GetCtxIsSignUp(ctx context.Context) bool {
	v, ok := ctx.Value(ctxKeyIsSignUp).(bool)
	if !ok {
		return false
	}
	return v
}

//SetCtxIsSignUp : set the flag indicating sign-up in the context
func SetCtxIsSignUp(ctx context.Context, v bool) context.Context {
	return context.WithValue(ctx, ctxKeyIsSignUp, v)
}

//SetCtxLogger : set the logger in the context
func SetCtxLogger(ctx context.Context, v *Logger) context.Context {
	return context.WithValue(ctx, ctxKeyLogger, v)
}

//GetCtxLogger : retrieve the logger from the context
func GetCtxLogger(ctx context.Context) *Logger {
	v, ok := ctx.Value(ctxKeyLogger).(*Logger)
	if !ok || v == nil {
		return nil
	}
	return v
}

//SetCtxMsg : set the message in the context
func SetCtxMsg(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, ctxKeyMsg, v)
}

//GetCtxMsg : retrieve the message from the context
func GetCtxMsg(ctx context.Context) string {
	v, ok := ctx.Value(ctxKeyMsg).(string)
	if !ok || v == "" {
		return ""
	}
	return v
}

//GetCtxProviderURLName : retrieve the provider url name from the context
func GetCtxProviderURLName(ctx context.Context) string {
	v, ok := ctx.Value(ctxKeyProviderURLName).(string)
	if !ok || v == "" {
		return ""
	}
	return v
}

//GetCtxPaymentID : retrieve the payment ID from the context
func GetCtxPaymentID(ctx context.Context) string {
	v, ok := ctx.Value(ctxKeyPaymentID).(string)
	if !ok || v == "" {
		return ""
	}
	return v
}

//SetCtxPaymentID : set the payment ID in the context
func SetCtxPaymentID(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, ctxKeyPaymentID, v)
}

//SetCtxProviderURLName : set the provider URL name in the context
func SetCtxProviderURLName(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, ctxKeyProviderURLName, v)
}

//GetCtxRequestID : retrieve the request ID from the context
func GetCtxRequestID(ctx context.Context) string {
	v, ok := ctx.Value(ctxKeyRequestID).(string)
	if !ok || v == "" {
		return ""
	}
	return v
}

//SetCtxRequestID : set the request ID in the context
func SetCtxRequestID(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID, v)
}

//GetCtxServiceID : retrieve the service ID from the context
func GetCtxServiceID(ctx context.Context) string {
	v, ok := ctx.Value(ctxKeyServiceID).(string)
	if !ok || v == "" {
		return ""
	}
	return v
}

//SetCtxStats : set the statistics data in the context
func SetCtxStats(ctx context.Context, v *ServerStatistics) context.Context {
	return context.WithValue(ctx, ctxKeyStats, v)
}

//GetCtxStats : retrieve the service ID from the context
func GetCtxStats(ctx context.Context) *ServerStatistics {
	v, ok := ctx.Value(ctxKeyStats).(*ServerStatistics)
	if !ok || v == nil {
		return nil
	}
	return v
}

//AddCtxStatsAPI : add API data to the statistics data in the context
func AddCtxStatsAPI(ctx context.Context, key ServerStatKey, label string, duration time.Duration) {
	stats := GetCtxStats(ctx)
	if stats == nil {
		return
	}
	stats.AddAPIStat(key, label, duration)
}

//AddCtxStatsData : add data to the statistics data in the context
func AddCtxStatsData(ctx context.Context, key ServerStatKey, data interface{}) {
	stats := GetCtxStats(ctx)
	if stats == nil {
		return
	}
	stats.AddData(key, data)
}

//AddCtxStatsTime : add time data to the statistics data in the context
func AddCtxStatsTime(ctx context.Context, key ServerStatKey, time time.Time) {
	stats := GetCtxStats(ctx)
	if stats == nil {
		return
	}
	stats.AddTime(key, time)
}

//AddCtxStatsCount : accumulate a count in the statistics data in the context
func AddCtxStatsCount(ctx context.Context, key ServerStatKey, count int) {
	stats := GetCtxStats(ctx)
	if stats == nil {
		return
	}
	stats.Count(key, count)
}

//SetCtxServiceID : set the service ID in the context
func SetCtxServiceID(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, ctxKeyServiceID, v)
}

//GetCtxTitleAlert : retrieve the alert title from the context
func GetCtxTitleAlert(ctx context.Context) string {
	v, ok := ctx.Value(ctxKeyTitleAlert).(string)
	if !ok || v == "" {
		return ""
	}
	return v
}

//SetCtxTitleAlert : set the notification title in the context
func SetCtxTitleAlert(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, ctxKeyTitleAlert, v)
}

//SetCtxURLShort : set the shortened URL in the context
func SetCtxURLShort(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, ctxKeyURLShort, v)
}

//GetCtxURLShort : retrieve the shortened URL from the context
func GetCtxURLShort(ctx context.Context) string {
	v, ok := ctx.Value(ctxKeyURLShort).(string)
	if !ok || v == "" {
		return ""
	}
	return v
}

//GetCtxUserID : retrieve the user ID from the context
func GetCtxUserID(ctx context.Context) *uuid.UUID {
	v, ok := ctx.Value(ctxKeyUserID).(*uuid.UUID)
	if !ok || v == nil {
		return nil
	}
	return v
}

//SetCtxUserID : set the user ID in the context
func SetCtxUserID(ctx context.Context, v *uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxKeyUserID, v)
}

//GetCtxTimeZone : retrieve the time zone from the context
func GetCtxTimeZone(ctx context.Context) string {
	v, ok := ctx.Value(ctxKeyTimeZone).(string)
	if !ok || v == "" {
		return ""
	}
	return v
}

//SetCtxTimeZone : set the time zone in the context
func SetCtxTimeZone(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, ctxKeyTimeZone, v)
}

//GetCtxType : retrieve the type from the context
func GetCtxType(ctx context.Context) string {
	v, ok := ctx.Value(ctxKeyType).(string)
	if !ok || v == "" {
		return ""
	}
	return v
}

//SetCtxType : set the type in the context
func SetCtxType(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, ctxKeyType, v)
}
