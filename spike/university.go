package spike

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	uctfirestore "github.com/tevjef/uct-backend/common/firestore"
	"github.com/tevjef/uct-backend/common/middleware"
	"github.com/tevjef/uct-backend/common/middleware/cache"
	"github.com/tevjef/uct-backend/common/middleware/httperror"
	"github.com/tevjef/uct-backend/common/model"
)

func universityHandler(expire time.Duration) gin.HandlerFunc {
	return cache.CachePage(func(c *gin.Context) {
		topicName := strings.ToLower(c.Param("topic"))
		firestore := uctfirestore.FromContext(c.Request.Context())

		if u, err := firestore.GetUniversity(c, topicName); err != nil {
			httperror.ServerError(c, err)
			return
		} else {
			response := model.Response{
				Data: &model.Data{University: u},
			}
			c.Set(middleware.ResponseKey, response)
		}
	}, expire)
}

func universitiesHandler(expire time.Duration) gin.HandlerFunc {
	return cache.CachePage(func(c *gin.Context) {
		firestore := uctfirestore.FromContext(c.Request.Context())

		if universities, err := firestore.GetUniversities(c.Request.Context()); err != nil {
			httperror.ServerError(c, err)
			return
		} else {
			response := model.Response{
				Data: &model.Data{Universities: universities},
			}
			c.Set(middleware.ResponseKey, response)
		}
	}, expire)
}
