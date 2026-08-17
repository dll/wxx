package handler

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

// GET /api/v1/student/profile
// 并发聚合：基础信息 + 五维孪生 + 性格 + 学业 + 竞赛 + 入党 + 社团 + 打卡 + 积分
// 错误容忍：单个子查询失败不影响整体，返回空数组/零值
func (h *StudentHandler) PersonalProfile(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	userID := userCtx.UserID
	result := gin.H{"user_id": userID, "username": userCtx.Username, "display_name": userCtx.DisplayName}
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 用匿名函数并发查询，写回 result
	query := func(key string, fn func() (interface{}, error)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			val, err := fn()
			if err != nil {
				log.Printf("聚合[%s]失败 user_id=%d: %v", key, userID, err)
				return
			}
			mu.Lock()
			result[key] = val
			mu.Unlock()
		}()
	}

	// 1. 基础信息（users 表）
	query("basic_info", func() (interface{}, error) {
		var college, major, className, enrollmentDate, enrollmentYear, status string
		err := h.db.QueryRow(
			"SELECT college, major, class_name, enrollment_date, enrollment_year, status FROM users WHERE id = ?",
			userID,
		).Scan(&college, &major, &className, &enrollmentDate, &enrollmentYear, &status)
		if err != nil {
			return nil, err
		}
		return gin.H{
			"college": college, "major": major, "class_name": className,
			"enrollment_date": enrollmentDate, "enrollment_year": enrollmentYear, "status": status,
		}, nil
	})

	// 2. 学业成绩汇总（student_grades）
	query("academic", func() (interface{}, error) {
		var courseCount int
		var credits, totalScore, gpa float64
		var passedCount, totalCount int
		err := h.db.QueryRow(
			"SELECT COUNT(*), COALESCE(SUM(credits_earned),0), COALESCE(AVG(score),0), "+
				"COALESCE(AVG(gpa),0), COALESCE(SUM(CASE WHEN passed=1 THEN 1 ELSE 0 END),0), COUNT(*) "+
				"FROM student_grades WHERE user_id = ?",
			fmt.Sprintf("%d", userID),
		).Scan(&courseCount, &credits, &totalScore, &gpa, &passedCount, &totalCount)
		if err != nil {
			return nil, err
		}
		passRate := 0.0
		if totalCount > 0 {
			passRate = float64(passedCount) / float64(totalCount) * 100
		}
		return gin.H{
			"course_count": courseCount, "total_credits": credits,
			"avg_score": totalScore, "avg_gpa": gpa, "pass_rate": passRate,
		}, nil
	})

	// 3. 竞赛报名（competition_registrations）
	query("competitions", func() (interface{}, error) {
		rows, err := h.db.Query(
			"SELECT competition_id, student_name, team_name, advisor_name, status, award_level FROM competition_registrations WHERE user_id = ? ORDER BY id DESC LIMIT 10",
			userID,
		)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var list []gin.H
		for rows.Next() {
			var cid int64
			var studentName, teamName, advisor, status, award string
			if err := rows.Scan(&cid, &studentName, &teamName, &advisor, &status, &award); err != nil {
				continue
			}
			list = append(list, gin.H{
				"competition_id": cid, "student_name": studentName, "team_name": teamName,
				"advisor_name": advisor, "status": status, "award_level": award,
			})
		}
		return list, nil
	})

	// 4. 入党进度（party_progress）
	query("party", func() (interface{}, error) {
		var currentStage, status string
		var applyDate string
		err := h.db.QueryRow(
			"SELECT current_stage, status, COALESCE(apply_date,'') FROM party_progress WHERE user_id = ? ORDER BY id DESC LIMIT 1",
			userID,
		).Scan(&currentStage, &status, &applyDate)
		if err != nil {
			if err == sql.ErrNoRows {
				return gin.H{"current_stage": "", "status": "", "apply_date": ""}, nil
			}
			return nil, err
		}
		return gin.H{"current_stage": currentStage, "status": status, "apply_date": applyDate}, nil
	})

	// 5. 社团参与（club_members）
	query("clubs", func() (interface{}, error) {
		rows, err := h.db.Query(
			"SELECT club_id, role, join_date FROM club_members WHERE user_id = ? AND (leave_date IS NULL OR leave_date = '') ORDER BY join_date DESC LIMIT 5",
			userID,
		)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var list []gin.H
		for rows.Next() {
			var cid int64
			var role, joinDate string
			if err := rows.Scan(&cid, &role, &joinDate); err != nil {
				continue
			}
			list = append(list, gin.H{"club_id": cid, "role": role, "join_date": joinDate})
		}
		return list, nil
	})

	// 6. 打卡记录（student_checkins）
	query("checkin", func() (interface{}, error) {
		var total int
		var lastDate string
		err := h.db.QueryRow(
			"SELECT COUNT(*), COALESCE(MAX(check_date),'') FROM student_checkins WHERE user_id = ?",
			userID,
		).Scan(&total, &lastDate)
		if err != nil {
			return nil, err
		}
		return gin.H{"total_days": total, "last_date": lastDate}, nil
	})

	// 7. 积分（student_points）
	query("points", func() (interface{}, error) {
		var total int
		err := h.db.QueryRow(
			"SELECT COALESCE(SUM(points),0) FROM student_points WHERE user_id = ?",
			userID,
		).Scan(&total)
		if err != nil {
			return nil, err
		}
		return gin.H{"total_points": total}, nil
	})

	wg.Wait()
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

// decodeJSON 解析 JSON 字符串字段（辅助）
func decodeJSON(raw string) []interface{} {
	var arr []interface{}
	if raw == "" {
		return arr
	}
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return arr
	}
	return arr
}

// UploadAvatar 上传/更新当前用户头像
// POST /api/v1/user/avatar  （multipart/form-data: file）
// 头像 base64 存 users.avatar_base64（SQLite 跨实例持久）
func (h *StudentHandler) UploadAvatar(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "未获取到上传文件"})
		return
	}
	defer file.Close()

	// 限制 3MB
	if header.Size > 3*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "头像文件超过 3MB 限制"})
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true}
	if !allowed[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "仅支持 png/jpg/webp/gif 图片"})
		return
	}

	bytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取文件失败"})
		return
	}

	mime := "image/png"
	switch ext {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".gif":
		mime = "image/gif"
	case ".webp":
		mime = "image/webp"
	}

	encoded := base64.StdEncoding.EncodeToString(bytes)
	if _, err := h.db.Exec(
		"UPDATE users SET avatar_base64 = ?, avatar_mime = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		encoded, mime, userCtx.UserID,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存头像失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0, "message": "头像已更新",
		"data": gin.H{"user_id": userCtx.UserID, "size": header.Size, "mime": mime},
	})
}

// ServeAvatar GET /api/v1/user/avatar/:user_id — 返回用户头像图片字节（base64 解码）
func (h *StudentHandler) ServeAvatar(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少 user_id"})
		return
	}

	var b64, mime string
	err := h.db.QueryRow(
		"SELECT COALESCE(avatar_base64,''), COALESCE(avatar_mime,'image/png') FROM users WHERE id = ?",
		userID,
	).Scan(&b64, &mime)
	if err != nil || b64 == "" {
		c.Status(http.StatusNotFound)
		return
	}

	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Header("Content-Type", mime)
	c.Data(http.StatusOK, mime, raw)
}
