package main

import (
	_ "net/http/pprof"
	"os"
	"time"
	"uct/common/conf"
	"uct/common/model"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"uct/julia/notifier"
	"uct/notification"
	"uct/redis"

	log "github.com/Sirupsen/logrus"
	"github.com/lib/pq"
	"github.com/pquerna/ffjson/ffjson"
	"golang.org/x/net/context"
)

type julia struct {
	app      *kingpin.ApplicationModel
	config   *juliaConfig
	redis    *redis.Helper
	notifier notifier.Notifier
	process  *Process
	ctx      context.Context
}

type juliaConfig struct {
	service        conf.Config
	inputFormat    string
	outputFormat   string
	daemonInterval time.Duration
	daemonFile     string
	scraperName    string
	scraperCommand string
}

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)
}

func main() {
	app := kingpin.New("julia", "An application that queue messages from the database")
	configFile := app.Flag("config", "configuration file for the application").Short('c').File()
	kingpin.MustParse(app.Parse(os.Args[1:]))

	// Parse configuration file
	config := conf.OpenConfigWithName(*configFile, app.Name)

	// Start profiling
	go model.StartPprof(config.DebugSever(app.Name))

	// Open connection to postgresql
	log.Infoln("Start monitoring PostgreSQL...")
	listener := pq.NewListener(config.DatabaseConfig(app.Name), 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.WithError(err).Fatalln("failure in listener")
		}
	})

	if err := listener.Listen("status_events"); err != nil {
		log.WithError(err).Fatalln("failed to listen on channel")
	}

	(&julia{
		app: app.Model(),
		config: &juliaConfig{
			service: config,
		},
		redis:    redis.NewHelper(config, app.Name),
		notifier: notifier.NewNotifier(listener),
		process: &Process{
			in:  make(chan model.UCTNotification),
			out: make(chan model.UCTNotification),
		},
		ctx: context.TODO(),
	}).init()
}

func (julia *julia) init() {
	go julia.process.Run(func(uctNotification model.UCTNotification) {
		log.WithFields(log.Fields{"topic": uctNotification.TopicName}).Infoln("queueing")
		if b, err := uctNotification.Marshal(); err != nil {
			log.WithError(err).Fatalln("failed to marshall notification")
		} else if _, err := julia.redis.Client.Set(notification.MainQueueData+uctNotification.TopicName, b, 0).Result(); err != nil {
			log.WithError(err).Warningln("failed to set notification data")
		} else if julia.redis.RPush(notification.MainQueue, uctNotification.TopicName); err != nil {
			log.WithError(err).Warningln("failed to push notification unto queue")
		}
	})

	for {
		waitForNotification(julia.ctx, julia.notifier, julia.process.Recv)
	}
}

func waitForNotification(ctx context.Context, l notifier.Notifier, onNotify func(notification *model.UCTNotification)) {
	for {
		select {
		case message, ok := <-l.Notify():
			if !ok {
				return
			}
			if message == "" {
				continue
			}

			go func() {
				var uctNotification model.UCTNotification
				if err := ffjson.UnmarshalFast([]byte(message), &uctNotification); err != nil {
					log.WithError(err).Errorln("failed to unmarsahll notification")
					return
				}

				onNotify(&uctNotification)
			}()

			// Received no notification from the database for 60 seconds.
		case <-time.After(1 * time.Minute):
			go l.Ping()
		case <-ctx.Done():
			return
		}
	}
}
