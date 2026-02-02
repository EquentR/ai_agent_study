package handler

import (
	"agent_study/internal/resp"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Register(apiGroup *gin.RouterGroup) {
	resp.HandlerWrapper(apiGroup, "hello",
		[]*resp.Handler{
			resp.NewJsonHandler(handleGet),
			resp.NewJsonHandler(handlePost),
			resp.NewJsonOptionsHandler(handleWithMiddleware),
		})
}

// handleGet
//
//	@Summary		handleGet
//	@Description	Get方法
//	@Tags			hello
//	@Produce		json
//	@Router			/hello/ [get]
//	@Security		ApiKeyAuth
//	@Success		200	{object}	resp.Result{data=string}
func handleGet() (string, string, resp.JsonResultWrapper, []resp.WrapperOption) {
	return http.MethodGet, "/", func(c *gin.Context) (any, error) {
		return "this is a get response", nil
	}, nil
}

// handlePost
//
//	@Summary		handlePost
//	@Description	Post方法
//	@Tags			hello
//	@Produce		json
//	@Router			/hello/post [post]
//	@Success		200	{object}	resp.Result{data=string}
func handlePost() (string, string, resp.JsonResultWrapper, []resp.WrapperOption) {
	return http.MethodPost, "/post", func(c *gin.Context) (any, error) {
		return "this is a post response", nil
	}, nil
}

// handleWithMiddleware
//
//	@Summary		handleWithMiddleware
//	@Description	带有中间件的方法
//	@Tags			hello
//	@Produce		json
//	@Router			/hello/middleware/{id} [get]
//	@Param			id	path		string	true	"一个ID"
//	@Success		200	{object}	resp.Result{data=string}
//	@Success		201	{object}	resp.Result{data=string}
//	@Fail			400	{object}
func handleWithMiddleware() (string, string, resp.JsonOptionsResultWrapper, []resp.WrapperOption) {
	return http.MethodGet, "/middleware/:id", func(c *gin.Context) (any, []resp.ResOpt, error) {
			id := c.Param("id")
			if id == "1" {
				return id, []resp.ResOpt{resp.WithCode(200)}, nil
			} else {
				return id, []resp.ResOpt{resp.WithCode(201)}, nil
			}
		}, []resp.WrapperOption{
			resp.WithMiddlewares(func(context *gin.Context) {
				id := context.Param("id")
				if id == "2" {
					context.AbortWithStatus(400)
				}
			}),
		}
}
