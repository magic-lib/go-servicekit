package mysqlchannel

import (
	"context"
	stdSQL "database/sql"
	"encoding/base64"
	"fmt"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
	driver "github.com/go-sql-driver/mysql"
	"github.com/hashicorp/go-multierror"
	"github.com/magic-lib/go-plat-retry/mysqlretry"
	"github.com/magic-lib/go-plat-utils/conv"
	"github.com/magic-lib/go-plat-utils/goroutines"
	"github.com/magic-lib/go-plat-utils/utils/httputil"
	comm "github.com/magic-lib/go-servicekit/watermill"
	cmap "github.com/orcaman/concurrent-map/v2"
	"log"
	"sync"
)

type Channel struct {
	sqlDb         *stdSQL.DB
	subscriberMap cmap.ConcurrentMap[string, *sql.Subscriber]
	namespace     string
	consumerGroup string
	publisher     *sql.Publisher
	handlers      cmap.ConcurrentMap[string, []comm.MessageHandler]
	subsMu        sync.RWMutex
	retryTimes    int
	retryService  *mysqlretry.RetryService
	errorHandler  func(msg *message.Message) error
}

func New(conf *driver.Config) (*Channel, error) {
	goChanTemp := new(Channel)
	err := goChanTemp.initMysqlDB(conf)
	if err != nil {
		return nil, err
	}
	goChanTemp.handlers = cmap.New[[]comm.MessageHandler]()
	goChanTemp.retryTimes = 3

	return goChanTemp, nil
}

func (g *Channel) Close() {
	for _, client := range g.subscriberMap.Items() {
		_ = client.Close()
	}
	_ = g.publisher.Close()
	_ = g.sqlDb.Close()

}

func (g *Channel) WithRetryTimes(retryTimes int) *Channel {
	g.retryTimes = retryTimes
	return g
}
func (g *Channel) WithNamespace(ns string) *Channel {
	g.namespace = ns
	return g
}
func (g *Channel) WithConsumerGroup(consumerGroup string) *Channel {
	g.consumerGroup = consumerGroup
	return g
}
func (g *Channel) WithErrorHandler(handler func(msg *message.Message) error) *Channel {
	g.errorHandler = handler
	return g
}

func (g *Channel) initMysqlDB(conf *driver.Config) error {
	if g.sqlDb != nil {
		return nil
	}

	if conf.Net == "" {
		conf.Net = "tcp"
	}
	if conf.DBName == "" {
		return fmt.Errorf("数据库连接失败, 数据库名称不能为空")
	}
	db, err := stdSQL.Open("mysql", conf.FormatDSN())
	if err != nil {
		return err
	}

	err = db.Ping()
	if err != nil {
		return err
	}

	g.sqlDb = db

	return nil
}
func (g *Channel) getSubscriber(consumerGroup string) (*sql.Subscriber, error) {
	if g.sqlDb == nil {
		return nil, fmt.Errorf("db error")
	}

	if g.consumerGroup == "" {
		g.consumerGroup = "default"
	}
	if consumerGroup == "" {
		consumerGroup = g.consumerGroup
	}

	if sub, ok := g.subscriberMap.Get(consumerGroup); ok {
		return sub, nil
	}

	logger := watermill.NewStdLogger(false, false)

	subscriber, err := sql.NewSubscriber(
		sql.BeginnerFromStdSQL(g.sqlDb),
		sql.SubscriberConfig{
			ConsumerGroup: consumerGroup,
			SchemaAdapter: sql.DefaultMySQLSchema{
				GenerateMessagesTableName: func(topic string) string {
					return g.getTableFromNamespace(topic)
				},
			},
			OffsetsAdapter: sql.DefaultMySQLOffsetsAdapter{
				GenerateMessagesOffsetsTableName: func(topic string) string {
					return g.getOffsetsTableFromNamespace(topic)
				},
			},
			InitializeSchema: true,
		},
		logger,
	)
	if err != nil {
		return nil, err
	}

	g.subscriberMap.Set(consumerGroup, subscriber)
	return subscriber, nil
}

func (g *Channel) getTableFromNamespace(topic string) string {
	if g.namespace == "" {
		g.namespace = "default"
	}
	return fmt.Sprintf("`%s_%s`", g.namespace, topic)
}
func (g *Channel) getOffsetsTableFromNamespace(topic string) string {
	if g.namespace == "" {
		g.namespace = "default"
	}
	return fmt.Sprintf("`%s_offsets_%s`", g.namespace, topic)
}
func (g *Channel) getPublisher() (*sql.Publisher, error) {
	if g.sqlDb == nil {
		return nil, fmt.Errorf("db error")
	}
	if g.publisher != nil {
		return g.publisher, nil
	}

	logger := watermill.NewStdLogger(false, false)

	publisher, err := sql.NewPublisher(
		sql.BeginnerFromStdSQL(g.sqlDb),
		sql.PublisherConfig{
			SchemaAdapter: sql.DefaultMySQLSchema{
				GenerateMessagesTableName: func(topic string) string {
					return g.getTableFromNamespace(topic)
				},
			},
		},
		logger,
	)
	if err != nil {
		return nil, err
	}

	g.publisher = publisher

	return publisher, nil
}

func (g *Channel) Publish(ctx context.Context, topic string, msg *message.Message) (string, error) {
	if msg == nil {
		return "", nil
	}
	publisher, err := g.getPublisher()
	if err != nil {
		return "", err
	}

	if msg.UUID == "" {
		msg.UUID = watermill.NewUUID()
	}
	base64Str := base64.StdEncoding.EncodeToString(msg.Payload)
	resp := httputil.CommResponse{
		Message: topic,
		Data:    base64Str,
	}
	mysqlPayload := []byte(conv.String(resp))
	msg = message.NewMessageWithContext(ctx, msg.UUID, mysqlPayload)
	if err = publisher.Publish(topic, msg); err != nil {
		return "", err
	}
	return msg.UUID, nil
}

func (g *Channel) Subscribe(topic string, handler comm.MessageHandler) error {
	g.subsMu.Lock()
	defer g.subsMu.Unlock()
	if oldHandlers, ok := g.handlers.Get(topic); ok {
		oldHandlers = append(oldHandlers, handler)
		g.handlers.Set(topic, oldHandlers)
		return nil
	}

	subscribe, err := g.getSubscriber(g.consumerGroup)
	if err != nil {
		return err
	}

	messages, err := subscribe.Subscribe(context.Background(), topic)
	if err != nil {
		return err
	}
	g.handlers.Set(topic, []comm.MessageHandler{handler})

	goroutines.GoAsync(func(params ...interface{}) {
		topicTemp := conv.String(params[0])
		for msg := range messages {
			g.dispatchMessages(topicTemp, msg)
		}
	}, topic)

	return nil
}

func getDataFromMsg(msg *message.Message) (*message.Message, error) {
	resp := new(httputil.CommResponse)
	_ = conv.Unmarshal(string(msg.Payload), resp)
	originData, err := base64.StdEncoding.DecodeString(conv.String(resp.Data))
	if err != nil {
		return nil, err
	}
	msg.Payload = originData
	return msg, nil
}

func (g *Channel) dispatchMessages(topic string, msg *message.Message) {
	allHandlers, ok := g.handlers.Get(topic)
	if !ok {
		log.Println("no handlers for topic: " + topic)
		return
	}
	var err error
	msg, err = getDataFromMsg(msg)
	if err != nil {
		return
	}

	var retError error
	for _, handler := range allHandlers {
		if err = handler(msg.UUID, string(msg.Payload)); err != nil {
			err = g.retryMessage(msg, handler, err)
			if err != nil {
				retError = multierror.Append(retError, err)
			}
		}
	}
	if retError != nil {
		log.Println("Subscribe handler error: ", retError.Error())
		msg.Ack()
		return
	}
	msg.Ack()
}

func (g *Channel) retryMessage(msg *message.Message, handler comm.MessageHandler, oldError error) error {
	if g.retryTimes <= 0 {
		return oldError
	}
	retryTimes := g.retryTimes

	var retError error
	for i := retryTimes; i > 0; i-- {
		if err := handler(msg.UUID, string(msg.Payload)); err != nil {
			retError = multierror.Append(retError, err)
		} else {
			return nil
		}
	}
	if retError == nil {
		return nil
	}

	//还是有错误，则需要执行错误的处理方式
	if g.errorHandler != nil {
		err := g.errorHandler(msg)
		if err == nil {
			return nil
		}
	}
	return g.mysqlRetryMessage(msg, handler, retError)
}
func (g *Channel) mysqlRetryMessage(msg *message.Message, handler comm.MessageHandler, retError error) error {
	if g.retryService == nil {
		mysqlRetryModel, err := mysqlretry.NewMysqlRetry(&mysqlretry.RetryConfig{
			Namespace: g.consumerGroup,
			TableName: g.namespace + "_retry_table",
			SqlDB:     g.sqlDb,
		})
		if err != nil {
			return retError
		}
		mysqlRetryModel.Start()
		g.retryService = mysqlRetryModel
	}
	_ = mysqlretry.Register(g.namespace, g.consumerGroup, func(param []any) (any, error) {
		msgId := conv.String(param[0])
		msgData := conv.String(param[1])
		if msgId != "" && msgData != "" {
			if err := handler(msg.UUID, string(msg.Payload)); err != nil {
				return "", err
			}
		}
		return msgId, nil
	})
	err := g.retryService.DoAsync(&mysqlretry.RetryRecord{
		RetryType: g.consumerGroup,
		Param: []any{
			msg.UUID, string(msg.Payload),
		},
	})
	if err != nil {
		return retError
	}
	return nil
}

func (g *Channel) SubscribeNew(topic string, consumerGroup string, handler comm.MessageHandler) error {
	subscribe, err := g.getSubscriber(consumerGroup)
	if err != nil {
		return err
	}

	messages, err := subscribe.Subscribe(context.Background(), topic)
	if err != nil {
		return err
	}

	goroutines.GoAsync(func(params ...interface{}) {
		for msg := range messages {
			msg, err = getDataFromMsg(msg)
			if err != nil {
				continue
			}
			if err = handler(msg.UUID, string(msg.Payload)); err != nil {
				err = g.retryMessage(msg, handler, err)
				if err == nil {
					msg.Ack()
					continue
				}
				log.Println("Subscribe handler error: ", err.Error())
				msg.Ack()
			} else {
				msg.Ack()
			}
		}
	})

	return nil
}
