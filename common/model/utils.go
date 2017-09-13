package model

import (
	"net"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/jmoiron/sqlx"
	"github.com/tevjef/uct-core/common/try"
	"gopkg.in/redis.v5"
)

func TimeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.WithFields(log.Fields{"elapsed": elapsed, "name": name}).Debug("")
}

func StartPprof(listener net.Listener) {
	log.Info(http.Serve(listener, nil))
}

func OpenPostgres(connection string) (database *sqlx.DB, err error) {
	err = try.DoWithOptions(func(attempt int) (retry bool, err error) {
		database, err = sqlx.Connect("postgres", connection)
		if err != nil {
			log.WithError(err).WithField("retry", attempt).Errorln("failed to open database connection")
			return true, err
		}

		return false, err
	}, &try.Options{BackoffStrategy: try.ExponentialJitterBackoff, MaxRetries: 5})

	return
}

func OpenRedis(addr, password string, database int, retry int) (client *redis.Client, err error) {
	err = try.DoWithOptions(func(attempt int) (retry bool, err error) {
		client = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       database})

		if err := client.Ping().Err(); err != nil {
			log.WithError(err).WithField("retry", attempt).Errorln("failed to open redis connection")
			return true, err
		}

		return false, err
	}, &try.Options{BackoffStrategy: try.ExponentialJitterBackoff, MaxRetries: retry})

	return
}
