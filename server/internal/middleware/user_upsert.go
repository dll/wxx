package middleware

import (
	"log"
	"net/http"

	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// UserUpserter 鐢ㄦ埛 upsert 鎺ュ彛锛堥伩鍏嶇洿鎺ヤ緷璧?repository 鍖咃級
type UserUpserter interface {
	UpsertFromContext(userCtx *model.UserContext) error
}

// EnsureUserExists 纭繚 JWT 涓殑鐢ㄦ埛瀛樺湪浜庢暟鎹簱锛圝IT 鍒涘缓锛?// 鐢ㄤ簬 Vercel 绛夋棤鏈嶅姟鍣ㄧ幆澧冿紝鍐峰惎鍔ㄦ椂鏁版嵁搴撲负绌轰絾 JWT 浠嶇劧鏈夋晥銆?func EnsureUserExists(upserter UserUpserter) gin.HandlerFunc {
	return func(c *gin.Context) {
		userCtx := GetUserContext(c)
		if userCtx == nil {
			c.Next()
			return
		}

		if err := upserter.UpsertFromContext(userCtx); err != nil {
			log.Printf("[EnsureUserExists] 鐢ㄦ埛 upsert 澶辫触 user=%s err=%v", userCtx.Username, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "鐢ㄦ埛鐘舵€佸紓甯革紝璇烽噸鏂扮櫥褰?,
			})
			return
		}

		c.Next()
	}
}
