package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tevjef/uct-core/common/model"
	"github.com/tevjef/uct-core/spike/middleware"
	"github.com/tevjef/uct-core/spike/middleware/cache"
	"github.com/tevjef/uct-core/spike/middleware/httperror"
	mtrace "github.com/tevjef/uct-core/spike/middleware/trace"
	"github.com/tevjef/uct-core/spike/store"
	"golang.org/x/net/context"
)

func subjectHandler(expire time.Duration) gin.HandlerFunc {
	return cache.CachePage(func(c *gin.Context) {
		subjectTopicName := strings.ToLower(c.ParamValue("topic"))

		if sub, _, err := SelectSubject(c, subjectTopicName); err != nil {
			if err == sql.ErrNoRows {
				httperror.NotFound(c, err)
				return
			}
			httperror.ServerError(c, err)
			return
		} else {
			response := model.Response{
				Data: &model.Data{Subject: &sub},
			}
			c.Set(middleware.ResponseKey, response)
		}
	}, expire)
}

func subjectsHandler(expire time.Duration) gin.HandlerFunc {
	return cache.CachePage(func(c *gin.Context) {
		season := strings.ToLower(c.ParamValue("season"))
		year := c.ParamValue("year")
		uniTopicName := strings.ToLower(c.ParamValue("topic"))

		if subjects, err := SelectSubjects(c, uniTopicName, season, year); err != nil {
			if err == sql.ErrNoRows {
				httperror.NotFound(c, err)
				return
			}
			httperror.ServerError(c, err)
			return
		} else {
			response := model.Response{
				Data: &model.Data{Subjects: subjects},
			}
			c.Set(middleware.ResponseKey, response)
		}
	}, expire)
}

func SelectSubject(ctx context.Context, subjectTopicName string) (subject model.Subject, b []byte, err error) {
	defer model.TimeTrack(time.Now(), "SelectProtoSubject")
	span := mtrace.NewSpan(ctx, "database.SelectSubject")
	span.SetLabel("topicName", subjectTopicName)
	defer span.Finish()

	m := map[string]interface{}{"topic_name": subjectTopicName}
	d := store.Data{}
	if err = store.Get(ctx, store.SelectProtoSubjectQuery, &d, m); err != nil {
		return
	}
	b = d.Data
	err = subject.Unmarshal(d.Data)
	return
}

func SelectSubjects(ctx context.Context, uniTopicName, season, year string) (subjects []*model.Subject, err error) {
	defer model.TimeTrack(time.Now(), "SelectSubjects")
	span := mtrace.NewSpan(ctx, "database.SelectSubjects")
	span.SetLabel("topicName", uniTopicName)
	span.SetLabel("year", year)
	span.SetLabel("season", season)
	defer span.Finish()

	m := map[string]interface{}{"topic_name": uniTopicName, "subject_season": season, "subject_year": year}
	err = store.Select(ctx, store.ListSubjectQuery, &subjects, m)
	if err == nil && len(subjects) == 0 {
		err = httperror.NoDataFound(fmt.Sprintf("No data subjects found for university=%s, season=%s, year=%s", uniTopicName, season, year))
	}
	return
}